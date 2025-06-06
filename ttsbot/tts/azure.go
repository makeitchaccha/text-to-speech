package tts

import (
	"context"
	"fmt"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/common"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
)

type AzureEngine struct {
	speechConfig      *speech.SpeechConfig
	speechSynthesizer *speech.SpeechSynthesizer
}

func NewAzureTTSEngine(speechConfig *speech.SpeechConfig) (Engine, error) {
	speechConfig.SetSpeechSynthesisOutputFormat(common.Audio48Khz96KBitRateMonoMp3)

	speechSynthesizer, err := speech.NewSpeechSynthesizerFromConfig(speechConfig, nil)
	if err != nil {
		return nil, err
	}

	return &AzureEngine{
		speechConfig:      speechConfig,
		speechSynthesizer: speechSynthesizer,
	}, nil
}

func (a *AzureEngine) Name() string {
	return "azure-speech-service"
}

func (a *AzureEngine) GenerateSpeech(ctx context.Context, request SpeechRequest) (*SpeechResponse, error) {
	a.speechConfig.SetSpeechSynthesisLanguage(request.LanguageCode)
	a.speechConfig.SetSpeechSynthesisVoiceName(request.VoiceName)

	outcomeChan := a.speechSynthesizer.SpeakTextAsync(request.Text)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case outcome := <-outcomeChan:
		defer outcome.Close()

		if outcome.Error != nil {
			return nil, outcome.Error
		}

		if outcome.Result.Reason != common.SynthesizingAudioCompleted {
			return nil, fmt.Errorf("failed to synthesize speech")
		}

		return &SpeechResponse{
			Format:       AudioFormatMp3,
			Channels:     1,
			AudioContent: outcome.Result.AudioData,
		}, nil
	}
}
