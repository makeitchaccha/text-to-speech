package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
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

type SpeechTask struct {
	Preset   preset.Preset
	Segments []string
}

type Session struct {
	engineRegistry *tts.EngineRegistry
	presetResolver preset.PresetResolver
	textChannelID  snowflake.ID
	conn           voice.Conn

	taskQueue  chan<- SpeechTask
	stopWorker chan struct{}
}

func New(engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, textChannelID snowflake.ID, conn voice.Conn) (*Session, error) {
	queue := make(chan SpeechTask, 10)
	stopWorker := make(chan struct{})
	session := &Session{
		engineRegistry: engineRegistry,
		presetResolver: presetResolver,
		textChannelID:  textChannelID,
		conn:           conn,
		taskQueue:      queue,
		stopWorker:     stopWorker,
	}

	go session.worker(queue, stopWorker)

	return session, nil
}

func (s *Session) Close(ctx context.Context) {
	s.conn.Close(ctx)
	close(s.stopWorker)
	close(s.taskQueue)
}

func (s *Session) worker(queue <-chan SpeechTask, stopWorker <-chan struct{}) {
	trackClose := make(chan struct{})
	audioQueue := make(chan []byte, 10)
	trackPlayer, err := newTrackPlayer(s.conn, audioQueue, trackClose)
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
			s.processTask(task, audioQueue)
		}
	}
}

func (s *Session) processTask(task SpeechTask, audioQueue chan<- []byte) {
	slog.Info("Processing speech task", "content", task.Segments, "preset", task.Preset.Identifier)

	for _, segment := range task.Segments {
		if segment == "" {
			slog.Warn("Skipping empty segment in speech task", "preset", task.Preset.Identifier)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		provider, err := s.performTextToSpeech(ctx, segment, task.Preset)
		if err != nil {
			slog.Error("Failed to perform text-to-speech", slog.Any("err", err), slog.String("content", segment))
			continue
		}

		slog.Info("Successfully synthesized speech for segment", "content", segment)
		audioQueue <- provider
	}
}

func (s *Session) performTextToSpeech(ctx context.Context, content string, preset preset.Preset) ([]byte, error) {
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

func (s *Session) enqueueSpeechTask(ctx context.Context, segments []string, preset preset.Preset) {
	if len(segments) == 0 {
		slog.Warn("No segments to process for TTS")
		return
	}

	task := SpeechTask{
		Preset:   preset,
		Segments: segments,
	}

	select {
	case s.taskQueue <- task:
		slog.Debug("Enqueued speech task", "segments", segments, "preset", preset.Identifier)
	case <-ctx.Done():
		slog.Warn("Context cancelled, not enqueuing task", "segments", segments, "preset", preset.Identifier)
	case <-s.stopWorker:
		slog.Warn("Session worker stopped, not enqueuing task", "segments", segments, "preset", preset.Identifier)
	default:
		slog.Warn("Task queue is full, dropping task", "segments", segments, "preset", preset.Identifier)
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

	// make the content safe and ready for TTS.
	content := event.Message.Content
	content = message.ReplaceUrlsWithPlaceholders(content)
	content = message.ConvertMarkdownToPlainText(content)
	content = message.LimitContentLength(content, 300)

	segments := make([]string, 0)
	segments = append(segments, member.EffectiveName())
	segments = append(segments, content)
	if nAttachment := len(event.Message.Attachments); nAttachment > 0 {
		segments = append(segments, fmt.Sprintf("%d attachments", nAttachment))
	}

	go func() {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		preset, err := s.presetResolver.Resolve(ctx, *event.GuildID, event.Message.Author.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		s.enqueueSpeechTask(ctx, segments, preset)
		slog.Info("Enqueued speech task", "content", content, "preset", preset.Identifier)
	}()
}

func (s *Session) onJoinVoiceChannel(event *events.GuildVoiceStateUpdate) {
	voiceState := event.VoiceState
	// notify someone joined the voice channel
	slog.Info("User joined voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	// TODO: remove hardcoded message
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		preset, err := s.presetResolver.Resolve(ctx, event.Member.GuildID, event.Member.User.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		segments := []string{
			event.Member.EffectiveName(),
			"がボイスチャンネルに参加しました",
		}

		s.enqueueSpeechTask(ctx, segments, preset)
	}()
}

func (s *Session) onLeaveVoiceChannel(event *events.GuildVoiceStateUpdate) LeaveResult {
	voiceState := event.OldVoiceState

	// notify someone left the voice channel
	slog.Info("User left voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	if isVoiceChannelEmpty(event.Client().Caches(), voiceState.GuildID, *voiceState.ChannelID, voiceState.UserID) {
		slog.Info("Voice channel is empty, closing session", "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)
		return LeaveResultClose
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		preset, err := s.presetResolver.Resolve(ctx, voiceState.GuildID, voiceState.UserID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		segments := []string{
			event.Member.EffectiveName(),
			"がボイスチャンネルから離脱しました",
		}

		s.enqueueSpeechTask(ctx, segments, preset)
	}()

	return LeaveResultKeepAlive
}

func isVoiceChannelEmpty(cache interface {
	cache.VoiceStateCache
	cache.MemberCache
}, guildID, channelID, ignoredUserID snowflake.ID) bool {
	empty := true
	cache.VoiceStatesForEach(guildID, func(voiceState discord.VoiceState) {
		// ignore voice states of the user who left the voice channel
		if voiceState.UserID == ignoredUserID {
			return
		}

		// ignore bot
		member, ok := cache.Member(guildID, voiceState.UserID)
		if ok && member.User.Bot {
			return
		}

		if voiceState.ChannelID != nil && *voiceState.ChannelID == channelID {
			empty = false
			return
		}
	})

	return empty
}

func (s *Session) String() string {
	return fmt.Sprintf("Session(textChannelID: %s, voiceChannelID: %s)", s.textChannelID, s.conn.ChannelID())
}
