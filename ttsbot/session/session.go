package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/disgoorg/audio/mp3"
	"github.com/disgoorg/audio/pcm"
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

type Session struct {
	engineRegistry *tts.EngineRegistry
	presetResolver preset.PresetResolver
	textChannelID  snowflake.ID
	conn           voice.Conn
}

func New(engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, textChannelID snowflake.ID, conn voice.Conn) (*Session, error) {
	session := &Session{
		engineRegistry: engineRegistry,
		presetResolver: presetResolver,
		textChannelID:  textChannelID,
		conn:           conn,
	}

	return session, nil
}

func (s *Session) Close(ctx context.Context) {
	s.conn.Close(ctx)
}

func (s *Session) requestTextToSpeech(ctx context.Context, content string, preset preset.Preset) {
	slog.Info("Request speech", "content", content)
	start := time.Now()
	engine, ok := s.engineRegistry.Get(preset.Engine)

	if !ok {
		slog.Error("TTS engine not found", slog.String("engine", preset.Engine), slog.String("content", content))
		return
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
		return
	}
	end := time.Now()
	slog.Info("Successfully synthesized speech", "duration", end.Sub(start))
	slog.Info("Playing audio in voice channel", "guildID", s.conn.GuildID(), "channelID", s.conn.ChannelID())

	provider, writer, err := mp3.NewCustomPCMFrameProvider(nil, 48000, 1)

	if err != nil {
		slog.Error("Failed to create MP3 provider", slog.Any("err", err))
		return
	}

	opusProvider, err := pcm.NewOpusProvider(nil, pcm.NewPCMFrameChannelConverterProvider(provider, 48000, 1, 2))

	if err != nil {
		slog.Error("Failed to create Opus provider", slog.Any("err", err))
		return
	}

	s.conn.SetOpusFrameProvider(opusProvider)

	reader := bytes.NewReader(audioConent)
	if _, err := io.Copy(writer, reader); err != nil {
		slog.Error("Failed to copy audio content to writer", slog.Any("err", err))
		return
	}

	slog.Info("Audio content copied to writer")
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
	content = message.ConvertMarkdownToPlainText(content)
	content = message.LimitContentLength(content, 300)
	content = message.AddMemberName(content, member.EffectiveName())
	content = message.AddAttachments(content, event.Message.Attachments)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		preset, err := s.presetResolver.Resolve(ctx, *event.GuildID, event.Message.Author.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		s.requestTextToSpeech(ctx, content, preset)
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

		content := event.Member.EffectiveName() + "がボイスチャンネルに参加しました"

		preset, err := s.presetResolver.Resolve(ctx, event.Member.GuildID, event.Member.User.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		s.requestTextToSpeech(ctx, content, preset)
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		content := event.Member.EffectiveName() + "がボイスチャンネルから退出しました"

		preset, err := s.presetResolver.Resolve(ctx, voiceState.GuildID, voiceState.UserID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		s.requestTextToSpeech(ctx, content, preset)
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
