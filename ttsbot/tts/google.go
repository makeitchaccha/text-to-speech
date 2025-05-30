package tts

import (
	"context"
	"log/slog"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

var audioConfig = texttospeechpb.AudioConfig{
	AudioEncoding:   texttospeechpb.AudioEncoding_MP3,
	SampleRateHertz: 48000,
}

var _ Engine = (*GoogleEngine)(nil)

// GoogleEngine is an implementation of the Engine interface for Google Text-to-Speech.
type GoogleEngine struct {
	client *texttospeech.Client
}

func NewGoogleTTSEngine(client *texttospeech.Client) *GoogleEngine {
	return &GoogleEngine{
		client: client,
	}
}

func (g *GoogleEngine) Name() string {
	return "google-cloud-text-to-speech"
}

func (g *GoogleEngine) GenerateSpeech(ctx context.Context, request SpeechRequest) ([]byte, error) {

	slog.Info("Synthesize speech", slog.String("text", request.Text))
	resp, err := g.client.SynthesizeSpeech(ctx, &texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: request.Text,
			},
		},
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: request.LanguageCode,
			Name:         request.VoiceName,
		},
		AudioConfig: &audioConfig,
	})

	if err != nil {
		slog.Error("failed to synthesize speech", "error", err)
		return nil, err
	}

	return resp.AudioContent, nil
}
