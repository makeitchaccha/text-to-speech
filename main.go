package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/handler"
	"github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	"github.com/makeitchaccha/text-to-speech/ttsbot"
	"github.com/makeitchaccha/text-to-speech/ttsbot/commands"
	"github.com/makeitchaccha/text-to-speech/ttsbot/session"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ttsClient, err := texttospeech.NewClient(ctx)
	if err != nil {
		slog.Error("Failed to create TTS client", slog.Any("err", err))
		os.Exit(-1)
	}

	var engine tts.Engine
	engine = tts.NewGoogleTTSEngine(ttsClient)

	if cfg.Redis.Enabled {
		slog.Info("Redis is enabled, setting up cache")
		slog.Info("Connecting to Redis")
		options, err := redis.ParseURL(cfg.Redis.Url)
		if err != nil {
			slog.Error("Failed to parse Redis URL", slog.Any("err", err))
			os.Exit(-1)
		}
		redisClient := redis.NewClient(options)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Error("Failed to connect to Redis", slog.Any("err", err))
			os.Exit(-1)
		}
		slog.Info("Connected to Redis")

		redisCache := cache.New(&cache.Options{
			Redis:      redisClient,
			LocalCache: cache.NewTinyLFU(5, time.Minute),
		})

		engine = tts.NewCachedTTSEngine(engine, redisCache, cfg.Redis.TTL, nil)
	} else {
		slog.Info("Redis is disabled, no cache will be used")
	}

	slog.Info("Syncing commands", slog.Bool("sync", *shouldSyncCommands))

	b := ttsbot.New(*cfg, Version, Commit)
	sessionManager := session.NewRouter()

	h := handler.New()
	// h.Command("/test", commands.TestHandler)
	// h.Autocomplete("/test", commands.TestAutocompleteHandler)
	h.Command("/join", commands.JoinHandler(engine, sessionManager))
	joinAutocompleteHanlder, err := commands.JoinAutocompleteHandler(ttsClient)
	if err != nil {
		slog.Error("Failed to create join autocomplete handler", slog.Any("err", err))
		os.Exit(-1)
	}
	h.Autocomplete("/join", joinAutocompleteHanlder)
	h.Command("/version", commands.VersionHandler(b))
	// h.Component("/test-button", components.TestComponent)

	if err = b.SetupBot(h, bot.NewListenerFunc(b.OnReady), sessionManager.CreateMessageHandler(), sessionManager.CreateVoiceStateHandler()); err != nil {
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
		if err = handler.SyncCommands(b.Client, commands.Commands, cfg.Bot.DevGuilds); err != nil {
			slog.Error("Failed to sync commands", slog.Any("err", err))
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
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
