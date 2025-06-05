package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

type LeaveResult int

// LeaveResult indicates which action to take after a user leaves the voice channel.
// LeaveResultKeepAlive means to keep the session alive, allowing it to continue processing messages.
// LeaveResultClose means to close the session, as there are no users left in the voice channel.
const (
	LeaveResultKeepAlive LeaveResult = iota
	LeaveResultClose
)

type Session struct {
	engineRegistry *tts.EngineRegistry
	presetResolver preset.PresetResolver
	textChannelID  snowflake.ID
	conn           voice.Conn
	voiceResources *i18n.VoiceResources
	textResource   *i18n.TextResource

	taskQueue  chan<- SpeechTask
	stopWorker chan struct{}
}

func New(engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, textChannelID snowflake.ID, conn voice.Conn, tr *i18n.TextResource, vrs *i18n.VoiceResources) (*Session, error) {
	queue := make(chan SpeechTask, 10)
	stopWorker := make(chan struct{})
	session := &Session{
		engineRegistry: engineRegistry,
		presetResolver: presetResolver,
		textChannelID:  textChannelID,
		conn:           conn,
		voiceResources: vrs,
		textResource:   tr,
		taskQueue:      queue,
		stopWorker:     stopWorker,
	}

	go session.worker(queue, stopWorker)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		preset, err := presetResolver.ResolveGuildPreset(ctx, conn.GuildID())
		if err != nil {
			slog.Error("Failed to resolve preset for session", slog.Any("err", err), slog.String("guildID", conn.GuildID().String()))
			return
		}

		vr, ok := vrs.GetOrGeneric(discord.Locale(preset.Language))
		if !ok {
			slog.Warn("Voice resources not found for locale", "locale", preset.Language)
			return
		}

		segments := []string{vr.Session.Launch}
		session.enqueueSpeechTask(ctx, NewSpeechTask(segments, preset))
	}()

	return session, nil
}

func (s *Session) Close(ctx context.Context) {
	s.conn.Close(ctx)
	close(s.stopWorker)
	close(s.taskQueue)
}

func (s *Session) worker(queue <-chan SpeechTask, stopWorker <-chan struct{}) {
	trackClose := make(chan struct{})
	audioQueue := make(chan *tts.SpeechResponse, 10)
	trackPlayer, err := newTrackPlayer(s.conn, audioQueue, trackClose)
	lastSpeakerID := snowflake.ID(0)
	s.conn.SetOpusFrameProvider(trackPlayer)
	if err != nil {
		slog.Error("Failed to create track player", slog.Any("err", err))
		return
	}
	slog.Info("Session worker started", "textChannelID", s.textChannelID, "voiceChannelID", s.conn.ChannelID())
	for {
		select {
		case <-stopWorker:
			slog.Info("Stopping session worker")
			return

		case task := <-queue:
			if task.ContainsSpeaker && task.SpeakerID != lastSpeakerID {
				task.Segments = append([]string{task.SpeakerName}, task.Segments...)
				lastSpeakerID = task.SpeakerID
			}
			s.processTask(task, audioQueue)
		}
	}
}

func (s *Session) processTask(task SpeechTask, audioQueue chan<- *tts.SpeechResponse) {
	slog.Info("Processing speech task", "content", task.Segments, "preset", task.Preset.Identifier)

	for _, segment := range task.Segments {
		if segment == "" {
			slog.Warn("Skipping empty segment in speech task", "preset", task.Preset.Identifier)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := s.performTextToSpeech(ctx, segment, task.Preset)
		if err != nil {
			slog.Error("Failed to perform text-to-speech", slog.Any("err", err), slog.String("content", segment))
			continue
		}

		slog.Info("Successfully synthesized speech for segment", "content", segment)
		audioQueue <- resp
	}
}

func (s *Session) performTextToSpeech(ctx context.Context, content string, preset preset.Preset) (*tts.SpeechResponse, error) {
	slog.Info("Request speech", "content", content)
	start := time.Now()
	engine, ok := s.engineRegistry.Get(preset.Engine)

	if !ok {
		slog.Error("TTS engine not found", slog.String("engine", preset.Engine), slog.String("content", content))
		return nil, fmt.Errorf("TTS engine %s not found", preset.Engine)
	}

	speechRequest := tts.SpeechRequest{
		Text:         content,
		LanguageCode: preset.Language,
		VoiceName:    preset.VoiceName,
		SpeakingRate: preset.SpeakingRate,
	}

	audioConent, err := engine.GenerateSpeech(ctx, speechRequest)

	if err != nil {
		slog.Error("Failed to synthesize speech", slog.Any("err", err), slog.String("content", content))
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}
	end := time.Now()
	slog.Info("Successfully synthesized speech", "duration", end.Sub(start))
	slog.Info("Playing audio in voice channel", "guildID", s.conn.GuildID(), "channelID", s.conn.ChannelID())

	return audioConent, nil
}

func (s *Session) enqueueSpeechTask(ctx context.Context, task SpeechTask) {
	if len(task.Segments) == 0 {
		slog.Warn("Skipping empty speech task", "preset", task.Preset.Identifier)
		return
	}

	slog := slog.With(slog.Attr{Key: "segments", Value: slog.AnyValue(task.Segments)}, slog.Attr{Key: "preset", Value: slog.StringValue(string(task.Preset.Identifier))})
	select {
	case <-ctx.Done():
		slog.Warn("Context cancelled, not enqueuing task")
		return
	case <-s.stopWorker:
		slog.Warn("Session worker stopped, not enqueuing task")
		return
	default:
	}

	select {
	case s.taskQueue <- task:
		slog.Debug("Enqueued speech task")
	default:
		slog.Warn("Task queue is full, dropping task")
	}
}

func (s *Session) onMessageCreate(event *events.MessageCreate) {
	// ignore messages from other channels or from bots
	if event.Message.Author.Bot {
		return
	}

	slog.Debug("Received message for TTS", "messageID", event.Message.ID, "content", event.Message.Content)

	member, err := event.Client().Rest().GetMember(*event.GuildID, event.Message.Author.ID)
	if err != nil {
		slog.Error("Failed to get member for message author", slog.Any("err", err), slog.String("userID", event.Message.Author.ID.String()))
		return
	}

	mentions := createIdToNameMap(event.Client(), *event.GuildID, event.Message.Mentions)

	// make the content safe and ready for TTS.
	content := event.Message.Content
	content = message.ReplaceUserMentions(content, mentions)
	content = message.ReplaceUrlsWithPlaceholders(content)
	content = message.ConvertMarkdownToPlainText(content)
	content = message.LimitContentLength(content, 300)

	segments := make([]string, 0)
	segments = append(segments, content)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		preset, err := s.presetResolver.Resolve(ctx, *event.GuildID, event.Message.Author.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		segments = func() []string {
			attachmentsCount := len(event.Message.Attachments)
			if attachmentsCount == 0 {
				return segments
			}
			vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(preset.Language))
			if !ok {
				slog.Warn("Voice resources not found for locale", "locale", preset.Language)
				return segments
			}
			// append the number of attachments to the segments
			attachmentsMessage := fmt.Sprintf(vr.Session.Attachments, attachmentsCount)
			return append(segments, attachmentsMessage)
		}()

		s.enqueueSpeechTask(ctx, NewSpeechTask(segments, preset, WithSpeaker(member.EffectiveName(), member.User.ID)))
		slog.Info("Enqueued speech task", "content", content, "preset", preset.Identifier)
	}()
}

func createIdToNameMap(client bot.Client, guildID snowflake.ID, users []discord.User) map[snowflake.ID]string {
	mentions := make(map[snowflake.ID]string, len(users))
	for _, user := range users {
		// we should fetch meber information to get the effective name
		// but to avoid unnecessary API calls, we can use the member cache.
		member, ok := client.Caches().Member(guildID, user.ID)
		if !ok {
			slog.Warn("Member not found in cache for mention", "mentionID", user.ID)
			mentions[user.ID] = user.EffectiveName()
		} else {
			mentions[user.ID] = member.EffectiveName()
		}
	}
	return mentions
}

func (s *Session) onJoinVoiceChannel(event *events.GuildVoiceStateUpdate) {
	voiceState := event.VoiceState
	// notify someone joined the voice channel
	slog.Info("User joined voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	// TODO: remove hardcoded message
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		preset, err := s.presetResolver.ResolveGuildPreset(ctx, event.Member.GuildID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(preset.Language))
		if !ok {
			slog.Warn("Voice resources not found for locale", "locale", preset.Language)
			return
		}
		segments := []string{
			fmt.Sprintf(vr.Session.UserJoin, event.Member.EffectiveName()),
		}

		s.enqueueSpeechTask(ctx, NewSpeechTask(segments, preset))
	}()
}

func (s *Session) onLeaveVoiceChannel(event *events.GuildVoiceStateUpdate) LeaveResult {
	voiceState := event.OldVoiceState

	// notify someone left the voice channel
	slog.Info("User left voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	if isVoiceChannelEmpty(event.Client().Rest(), event.Client().Caches(), voiceState.GuildID, *voiceState.ChannelID, voiceState.UserID) {
		slog.Info("Voice channel is empty, closing session", "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)
		return LeaveResultClose
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		preset, err := s.presetResolver.ResolveGuildPreset(ctx, event.Member.GuildID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(preset.Language))
		if !ok {
			slog.Warn("Voice resources not found for locale", "locale", preset.Language)
			return
		}
		segments := []string{
			fmt.Sprintf(vr.Session.UserLeave, event.Member.EffectiveName()),
		}

		s.enqueueSpeechTask(ctx, NewSpeechTask(segments, preset))
	}()

	return LeaveResultKeepAlive
}

func isVoiceChannelEmpty(
	client rest.Users,
	cache interface {
		cache.VoiceStateCache
		cache.MemberCache
	}, guildID, channelID, ignoredUserID snowflake.ID,
) bool {
	empty := true
	cache.VoiceStatesForEach(guildID, func(voiceState discord.VoiceState) {
		// ignore voice states of the user who left the voice channel
		if voiceState.UserID == ignoredUserID {
			return
		}

		// ignore bot

		user, ok := fetchUser(guildID, voiceState.UserID, client, cache)
		if !ok {
			slog.Warn("Failed to fetch user for voice state", "userID", voiceState.UserID, "guildID", guildID)
			return
		}

		if user.Bot {
			slog.Debug("Ignoring bot in voice channel", "userID", voiceState.UserID, "guildID", guildID)
			return
		}

		if voiceState.ChannelID != nil && *voiceState.ChannelID == channelID {
			empty = false
			return
		}
	})

	return empty
}

func fetchUser(guildID, userID snowflake.ID, client rest.Users, cache cache.MemberCache) (*discord.User, bool) {
	member, ok := cache.Member(guildID, userID)
	if ok {
		return &member.User, true
	}

	user, err := client.GetUser(userID)
	if err == nil {
		return user, true
	}

	return nil, false
}

func (s *Session) String() string {
	return fmt.Sprintf("Session(textChannelID: %s, voiceChannelID: %s)", s.textChannelID, s.conn.ChannelID())
}
