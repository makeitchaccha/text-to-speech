package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/makeitchaccha/text-to-speech/ttsbot"
	"github.com/makeitchaccha/text-to-speech/ttsbot/commands"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
	"github.com/makeitchaccha/text-to-speech/ttsbot/session"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	trs, err := i18n.LoadTextResources("./locales/text/", "en-US")
	if err != nil {
		slog.Error("Failed to load text resources", slog.Any("err", err))
		os.Exit(-1)
	}
	vrs, err := i18n.LoadVoiceResources("./locales/voice/")
	if err != nil {
		slog.Error("Failed to load voice resources", slog.Any("err", err))
		os.Exit(-1)
	}

	shouldSyncCommands := flag.Bool("sync-commands", false, "Whether to sync commands to discord")
	path := flag.String("config", "config.toml", "path to config")
	flag.Parse()

	cfg, err := ttsbot.LoadConfig(*path)
	if err != nil {
		slog.Error("Failed to read config", slog.Any("err", err))
		os.Exit(-1)
	}

	setupLogger(cfg.Log)
	slog.Info("Starting ttsbot...", slog.String("version", Version), slog.String("commit", Commit))
	slog.Info("Connecting to Google Cloud TTS")

	slog.Info("Syncing commands", slog.Bool("sync", *shouldSyncCommands))

	b := ttsbot.New(*cfg, Version, Commit)

	opts := make([]engineOpt, 0)
	var redisClient *redis.Client
	if cfg.Redis.Enabled {
		slog.Info("Connecting to Redis", slog.String("url", cfg.Redis.Url))
		option, err := redis.ParseURL(cfg.Redis.Url)
		if err != nil {
			slog.Error("Failed to parse Redis URL", slog.Any("err", err))
			os.Exit(-1)
		}

		redisClient = redis.NewClient(option)
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			slog.Error("Failed to connect to Redis", slog.Any("err", err))
			os.Exit(-1)
		}

		slog.Info("Connected to Redis", slog.String("url", cfg.Redis.Url))

		opts = append(opts, withCache(cache.New(&cache.Options{
			Redis:      redisClient,
			LocalCache: cache.NewTinyLFU(10, 5*time.Minute),
		}), cfg.Redis.TTL))
	}

	sessionManager := session.NewSessionManager()

	engineRegistry := tts.NewEngineRegistry()
	registerDefaultEngines(engineRegistry, opts...)

	presetRegistry := preset.NewPresetRegistry()
	for identifier, presetConfig := range cfg.Presets {
		if err := registerPreset(engineRegistry, presetRegistry, identifier, presetConfig); err != nil {
			slog.Error("Failed to register preset", slog.String("identifier", identifier), slog.Any("err", err))
			os.Exit(-1)
		}
	}

	dialector, err := resolveDialector(cfg.Database)
	if err != nil {
		slog.Error("Failed to resolve database dialector", slog.Any("err", err))
		os.Exit(-1)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		slog.Error("Failed to connect to database", slog.Any("err", err))
		os.Exit(-1)
	}

	presetResolver, err := preset.NewPresetResolver(presetRegistry, preset.NewPresetIDRepository(db), preset.PresetID(cfg.Bot.FallbackPresetID))
	if err != nil {
		slog.Error("Failed to create preset resolver", slog.Any("err", err))
		os.Exit(-1)
	}

	h := handler.New()
	h.Command("/join", commands.JoinHandler(engineRegistry, presetResolver, sessionManager, trs, vrs))
	if err != nil {
		slog.Error("Failed to create join autocomplete handler", slog.Any("err", err))
		os.Exit(-1)
	}
	h.Command("/leave", commands.LeaveHandler(sessionManager, trs))
	h.Command("/preset", commands.PresetHandler(presetRegistry, presetResolver, preset.NewPresetIDRepository(db), trs))
	h.Command("/version", commands.VersionHandler(b))

	sessionRestorationListener := createSessionRestorationListener(redisClient, engineRegistry, presetResolver, sessionManager, trs, vrs)

	listeners := []bot.EventListener{
		h,
		bot.NewListenerFunc(b.OnReady),
		sessionManager.CreateMessageHandler(),
		sessionManager.CreateVoiceStateHandler(),
		sessionRestorationListener,
	}

	if err = b.SetupBot(listeners...); err != nil {
		slog.Error("Failed to setup bot", slog.Any("err", err))
		os.Exit(-1)
	}

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		b.Client.Close(ctx)
	}()

	if *shouldSyncCommands {
		slog.Info("Syncing commands", slog.Any("guild_ids", cfg.Bot.DevGuilds))
		if err = handler.SyncCommands(b.Client, commands.Commands(trs), cfg.Bot.DevGuilds); err != nil {
			slog.Error("Failed to sync commands", slog.Any("err", err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = b.Client.OpenGateway(ctx); err != nil {
		slog.Error("Failed to open gateway", slog.Any("err", err))
		os.Exit(-1)
	}

	slog.Info("Bot is running. Press CTRL-C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
	slog.Info("Shutting down bot...")
}

func setupLogger(cfg ttsbot.LogConfig) {
	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Level,
	}

	var sHandler slog.Handler
	switch cfg.Format {
	case "json":
		sHandler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		sHandler = slog.NewTextHandler(os.Stdout, opts)
	default:
		slog.Error("Unknown log format", slog.String("format", cfg.Format))
		os.Exit(-1)
	}
	slog.SetDefault(slog.New(sHandler))
}

type engineOpt func(tts.Engine) tts.Engine

func withCache(redisCache *cache.Cache, ttl time.Duration) engineOpt {
	return func(e tts.Engine) tts.Engine {
		return tts.NewCachedTTSEngine(e, redisCache, ttl, nil)
	}
}

func applyEngineOpts(engine tts.Engine, opts ...engineOpt) tts.Engine {
	for _, opt := range opts {
		engine = opt(engine)
	}
	return engine
}

func registerDefaultEngines(registry *tts.EngineRegistry, opts ...engineOpt) error {
	googleEngine, err := prepareGoogleTTSEngine()
	if err != nil {
		slog.Error("Failed to prepare Google TTS engine", slog.Any("err", err))
		return err
	}

	registry.Register("google", applyEngineOpts(googleEngine, opts...))
	slog.Info("Default TTS engines registered")
	return nil
}

func prepareGoogleTTSEngine() (tts.Engine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ttsClient, err := texttospeech.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return tts.NewGoogleTTSEngine(ttsClient), nil
}

func registerPreset(engineRegistry *tts.EngineRegistry, presetRegistry *preset.PresetRegistry, identifier string, presetConfig ttsbot.PresetConfig) error {
	if presetConfig.Engine == "" {
		return fmt.Errorf("preset %s does not have an engine specified", identifier)
	}

	_, ok := engineRegistry.Get(presetConfig.Engine)
	if !ok {
		return fmt.Errorf("preset %s references unknown engine %s", identifier, presetConfig.Engine)
	}

	preset := preset.Preset{
		Identifier: preset.PresetID(identifier),
		Engine:     presetConfig.Engine,
		Language:   presetConfig.Language,
		VoiceName:  presetConfig.VoiceName,
	}
	if err := presetRegistry.Register(preset); err != nil {
		return err
	}

	slog.Info("Registered preset", "preset", identifier, "engine", presetConfig.Engine, "language", presetConfig.Language, "voiceName", presetConfig.VoiceName)
	return nil
}

func resolveDialector(cfg ttsbot.DatabaseConfig) (gorm.Dialector, error) {
	switch cfg.Driver {
	case "sqlite3":
		return sqlite.Open(cfg.Dsn), nil
	case "mysql":
		return mysql.Open(cfg.Dsn), nil
	case "postgres":
		return postgres.Open(cfg.Dsn), nil
	}
	return nil, fmt.Errorf("unknown database driver: %s", cfg.Driver)
}

func createSessionRestorationListener(redisClient *redis.Client, engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, sessionManager session.SessionManager, trs *i18n.TextResources, vrs *i18n.VoiceResources) bot.EventListener {
	return bot.NewListenerFunc(func(r *events.Ready) {
		slog.Info("Restoring sessions from persistence")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		heartbeatInterval := 30 * time.Second
		persistenceManager := session.NewPersistenceManager(r.Application.ID, redisClient, heartbeatInterval)

		persistenceManager.StartHeartbeatLoop()
		sessionManager.AddObserver(persistenceManager)
		persistenceManager.Restore(ctx, sessionManager, func(guildID, voiceChannelID, readingChannelID snowflake.ID) (*session.Session, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			conn := r.Client().VoiceManager().GetConn(guildID)
			if conn == nil {
				conn = r.Client().VoiceManager().CreateConn(guildID)
			}

			err := conn.Open(ctx, voiceChannelID, false, true)
			if err != nil {
				slog.Error("Failed to open voice connection", slog.Any("err", err), slog.String("guildID", guildID.String()), slog.String("voiceChannelID", voiceChannelID.String()))
				return nil, err
			}

			// we may not use fallback but there is no way to get the text resource from the session currently.
			// however, it is just fallback, so it does not matter much.
			tr := trs.GetFallback()
			session, err := session.New(engineRegistry, presetResolver, readingChannelID, conn, &tr, vrs)
			if err != nil {
				slog.Error("Failed to create session from persistence", slog.Any("err", err), slog.String("readingChannelID", readingChannelID.String()))
				return nil, err
			}

			slog.Info("Restored session from persistence", slog.String("readingChannelID", readingChannelID.String()), slog.String("voiceChannelID", voiceChannelID.String()))
			return session, nil
		})

		slog.Info("Persistence manager started", slog.String("applicationID", r.Application.ID.String()), slog.Duration("heartbeatInterval", heartbeatInterval))
	})
}
