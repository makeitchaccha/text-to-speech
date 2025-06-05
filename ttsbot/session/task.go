package session

import (
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
)

// SpeechTask represents a task for text-to-speech processing.
type SpeechTask struct {
	Segments []string
	Preset   preset.Preset

	// option: with speaker?
	ContainsSpeaker bool
	SpeakerName     string
	SpeakerID       snowflake.ID
}

type SpeechTaskOpt func(s *SpeechTask)

func NewSpeechTask(segments []string, preset preset.Preset, opts ...SpeechTaskOpt) SpeechTask {
	task := &SpeechTask{
		Segments: segments,
		Preset:   preset,
	}
	task.apply(opts...)
	return *task
}

func (s *SpeechTask) apply(opts ...SpeechTaskOpt) {
	for _, opt := range opts {
		opt(s)
	}
}

func WithSpeaker(speakerName string, speakerID snowflake.ID) SpeechTaskOpt {
	return func(s *SpeechTask) {
		s.ContainsSpeaker = true
		s.SpeakerName = speakerName
		s.SpeakerID = speakerID
	}
}
