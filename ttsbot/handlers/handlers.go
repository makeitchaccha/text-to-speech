package handlers

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/disgoorg/audio/mp3"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
)

var (
	audioConfig = texttospeechpb.AudioConfig{
		AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
		SampleRateHertz: 48000,
	}
)

type sessionConfig struct {
	language   string
	ssmlGender texttospeechpb.SsmlVoiceGender
	voiceName  string
}

func (c *sessionConfig) voiceSelectionParams() *texttospeechpb.VoiceSelectionParams {
	return &texttospeechpb.VoiceSelectionParams{
		LanguageCode: c.language,
		SsmlGender:   c.ssmlGender,
		Name:         c.voiceName,
	}
}

func (c *sessionConfig) apply(opts []SessionConfigOpt) {
	for _, opt := range opts {
		opt(c)
	}
}

type SessionConfigOpt func(sessionConfig *sessionConfig)

func WithLanguage(language string) SessionConfigOpt {
	return func(sessionConfig *sessionConfig) {
		sessionConfig.language = language
	}
}

func WithSsmlGender(ssmlGender texttospeechpb.SsmlVoiceGender) SessionConfigOpt {
	return func(sessionConfig *sessionConfig) {
		sessionConfig.ssmlGender = ssmlGender
	}
}

func WithVoiceName(voiceName string) SessionConfigOpt {
	return func(sessionConfig *sessionConfig) {
		sessionConfig.voiceName = voiceName
	}
}

func SessionMessageHandler(textChannelID snowflake.ID, conn voice.Conn, ttsClient *texttospeech.Client, opts ...SessionConfigOpt) bot.EventListener {
	config := &sessionConfig{
		language:   "en-US",
		ssmlGender: texttospeechpb.SsmlVoiceGender_SSML_VOICE_GENDER_UNSPECIFIED,
	}
	config.apply(opts)

	return bot.NewListenerFunc(func(event *events.MessageCreate) {
		// ignore messages from other channels or from bots
		if event.ChannelID != textChannelID || event.Message.Author.Bot {
			return
		}

		slog.Debug("Received message for TTS", "messageID", event.Message.ID, "content", event.Message.Content)

		// make the content safe and ready for TTS.
		content := event.Message.Content
		content = message.LimitLength(content, 200)
		content = message.AddAttachments(content, event.Message.Attachments)

		go func() {
			slog.Debug("Synthesize speech", "messageID", event.Message.ID)
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resp, err := ttsClient.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
				Input: &texttospeechpb.SynthesisInput{
					InputSource: &texttospeechpb.SynthesisInput_Text{
						Text: content,
					},
				},
				Voice:       config.voiceSelectionParams(),
				AudioConfig: &audioConfig,
			})

			if err != nil {
				slog.Error("Failed to synthesize speech", "messageID", event.Message.ID, slog.Any("err", err), slog.String("content", content))
				return
			}
			end := time.Now()
			slog.Debug("Successfully synthesized speech", "messageID", event.Message.ID, "duration", end.Sub(start))
			slog.Debug("Playing audio in voice channel", "messageID", event.Message.ID, "guildID", conn.GuildID(), "channelID", conn.ChannelID())

			provider, writer, err := mp3.NewCustomPCMFrameProvider(nil, 48000, 1)

			if err != nil {
				slog.Error("Failed to create MP3 provider", "messageID", event.Message.ID, slog.Any("err", err))
				return
			}

			opusProvider, err := pcm.NewOpusProvider(nil, pcm.NewPCMFrameChannelConverterProvider(provider, 48000, 1, 2))

			if err != nil {
				slog.Error("Failed to create Opus provider", "messageID", event.Message.ID, slog.Any("err", err))
				return
			}

			conn.SetOpusFrameProvider(opusProvider)

			reader := bytes.NewReader(resp.AudioContent)
			if _, err := io.Copy(writer, reader); err != nil {
				slog.Error("Failed to copy audio content to writer", "messageID", event.Message.ID, slog.Any("err", err))
				return
			}

			slog.Debug("Audio content copied to writer", "messageID", event.Message.ID)
		}()
	})
}
