package audio

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

func NewSpeechTask(preset preset.Preset, segments []string, opts ...TaskOption) SpeechTask {
	task := &SpeechTask{
		Preset:   preset,
		Segments: segments,
	}
	task.apply(opts)
	return *task
}

type SpeechTask struct {
	Preset   preset.Preset
	Segments []string

	ContainsName bool
	Speaker      snowflake.ID
	SpeakerName  string
}

func (t *SpeechTask) apply(opts []TaskOption) {
	for _, opt := range opts {
		opt(t)
	}
}

type TaskOption func(*SpeechTask)

func WithSpeaker(speakerID snowflake.ID, speakerName string) TaskOption {
	return func(task *SpeechTask) {
		task.ContainsName = true
		task.Speaker = speakerID
		task.SpeakerName = speakerName
	}
}

type AudioWorker interface {
	Start() error
	Stop() error
	EnqueueTask(task SpeechTask) error
}

var _ AudioWorker = (*audioWorkerImpl)(nil)

type audioWorkerImpl struct {
	engineRegistry *tts.EngineRegistry

	taskQueue   chan SpeechTask
	stopWorker  chan struct{}
	trackPlayer *trackPlayer
}

func NewAudioWorker(conn voice.Conn, engineRegistry *tts.EngineRegistry, queueSize int) (AudioWorker, error) {
	queue := make(chan SpeechTask, queueSize)
	stop := make(chan struct{})

	// we estimate that the track player will need to handle 3 times the queue size
	// because ordinary speech tasks contain multiple segments, like {{speker_name}}, {{body}}, {{extra: like "1 attachment"}}
	// so we multiply the queue size by 3 to ensure we have enough capacity
	// to handle the segments without blocking
	trackPlayer, err := newTrackPlayer(conn, stop, queueSize*3)
	if err != nil {
		slog.Error("failed to create track player", "error", err)
		return nil, err
	}

	worker := &audioWorkerImpl{
		engineRegistry: engineRegistry,
		taskQueue:      queue,
		stopWorker:     stop,
		trackPlayer:    trackPlayer,
	}
	conn.SetOpusFrameProvider(trackPlayer)

	return worker, nil
}

func (w *audioWorkerImpl) Start() error {
	slog.Info("AudioWorker started")
	go w.workerLoop()
	return nil
}

func (w *audioWorkerImpl) Stop() error {
	slog.Info("Stopping AudioWorker")
	close(w.stopWorker)
	if w.trackPlayer != nil {
		w.trackPlayer.Close()
	}
	slog.Info("AudioWorker stopped")
	return nil
}

// Non-blocking enqueue method
func (w *audioWorkerImpl) EnqueueTask(task SpeechTask) error {
	select {
	case <-w.stopWorker:
		slog.Warn("Audio worker is stopped, cannot enqueue task")
		return fmt.Errorf("audio worker is stopped, cannot enqueue task for preset %s with segments %v", task.Preset.Identifier, task.Segments)
	default:
		// no-op, continue to enqueue the task
	}

	select {
	case w.taskQueue <- task:
		slog.Info("Enqueued speech task", "preset", task.Preset.Identifier, "segments", task.Segments)
		return nil
	default:
		slog.Warn("Failed to enqueue speech task, queue is full", "preset", task.Preset.Identifier, "segments", task.Segments)
		return fmt.Errorf("failed to enqueue speech task for preset %s with segments %v, queue is full", task.Preset.Identifier, task.Segments)
	}
}

func (w *audioWorkerImpl) workerLoop() {
	lastSpeakerID := snowflake.ID(0)
	for {
		select {
		case <-w.stopWorker:
			slog.Info("Audio worker stopped")
			return
		case task := <-w.taskQueue:
			// If the task is from a different speaker, add speaker name to the track
			if task.ContainsName && task.Speaker != lastSpeakerID {
				task.Segments = append([]string{task.SpeakerName}, task.Segments...)
				lastSpeakerID = task.Speaker
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := w.processTask(ctx, task); err != nil {
				slog.Error("Failed to process speech task", "error", err, "task", task)
			}
			cancel()
		}
	}
}

func (w *audioWorkerImpl) processTask(ctx context.Context, task SpeechTask) error {
	engine, ok := w.engineRegistry.Get(task.Preset.Engine)
	if !ok {
		slog.Error("TTS engine not found", "engine", task.Preset.Engine)
		return fmt.Errorf("failed to get TTS engine %s", task.Preset.Engine)
	}

	for _, segment := range task.Segments {
		audioContent, err := engine.GenerateSpeech(ctx, tts.SpeechRequest{
			Text:         segment,
			LanguageCode: task.Preset.Language,
			VoiceName:    task.Preset.VoiceName,
			SpeakingRate: task.Preset.SpeakingRate,
		})
		if err != nil {
			slog.Error("Failed to generate speech", "error", err, "segment", segment)
			return fmt.Errorf("failed to generate speech for segment %s: %w", segment, err)
		}

		err = w.trackPlayer.ProvideAudio(audioContent)
		if err != nil {
			slog.Warn("Failed to provide audio to track player", "error", err, "segment", segment)
		}
	}

	slog.Info("Processed speech task", "preset", task.Preset.Identifier, "segments", task.Segments)
	return nil
}
