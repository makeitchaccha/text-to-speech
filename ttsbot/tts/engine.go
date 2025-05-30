package tts

import (
	"context"
)

// Engine is a generic interface for text-to-speech engines.
// It can be implemented by various TTS engines to provide a unified interface for text-to-speech operations.
// However, currenlty it is only implemented by the Google TTS engine so SynthetizeRequest leaks some Google TTS specific parameters.
//
// FIXME: when other TTS engines are implemented, this interface should be made more generic.
// Yet, it is not clear how to make it generic, since different TTS engines have different parameters.
type Engine interface {
	// Name returns the name of the TTS engine, e.g. "Google TTS", "Azure TTS", etc.
	Name() string

	// GenerateSpeech generates speech from the given text and returns the mp3 audio data.
	GenerateSpeech(ctx context.Context, request SpeechRequest) (audioContent []byte, err error)
}

type SpeechRequest struct {
	Text         string
	LanguageCode string
	VoiceName    string
}
