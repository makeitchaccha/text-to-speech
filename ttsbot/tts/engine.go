package tts

import (
	"context"
)

// Engine is a generic interface for text-to-speech engines.
// It can be implemented by various TTS engines to provide a unified interface for text-to-speech operations.
// However, currently it is only implemented by the Google TTS engine so SynthetizeRequest leaks some Google TTS specific parameters.
//
// FIXME: when other TTS engines are implemented, this interface should be made more generic.
// Yet, it is not clear how to make it generic, since different TTS engines have different parameters.
type Engine interface {
	// Name returns the name of the TTS engine, e.g. "Google TTS", "Azure TTS", etc.
	Name() string

	// GenerateSpeech generates speech from the given text and returns the audio content.
	GenerateSpeech(ctx context.Context, request SpeechRequest) (resp *SpeechResponse, err error)
}

type (
	SpeechRequest struct {
		Text         string
		LanguageCode string
		VoiceName    string
		SpeakingRate float64
	}

	AudioFormat int

	SpeechResponse struct {
		Format       AudioFormat
		Channels     int
		AudioContent []byte
	}
)

const (
	AudioFormatUnknown AudioFormat = iota
	AudioFormatMp3
)

type EngineRegistry struct {
	engines map[string]Engine // identifier -> Engine
}

func NewEngineRegistry() *EngineRegistry {
	return &EngineRegistry{
		engines: make(map[string]Engine),
	}
}

func (r *EngineRegistry) Register(identifier string, engine Engine) {
	if _, exists := r.engines[identifier]; exists {
		panic("engine already registered: " + identifier)
	}
	r.engines[identifier] = engine
}

func (r *EngineRegistry) Get(identifier string) (Engine, bool) {
	engine, ok := r.engines[identifier]
	return engine, ok
}

func (r *EngineRegistry) MustGet(identifier string) Engine {
	engine, ok := r.Get(identifier)
	if !ok {
		panic("engine not found: " + identifier)
	}
	return engine
}
