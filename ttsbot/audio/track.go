package audio

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"

	"github.com/disgoorg/audio"
	"github.com/disgoorg/audio/mp3"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/voice"
)

type trackPlayer struct {
	audio.Player
	queue      chan []byte
	conn       voice.Conn
	provider   pcm.FrameProvider
	stopSignal <-chan struct{}
}

func newTrackPlayer(conn voice.Conn, stopSignal <-chan struct{}, queueSize int) (*trackPlayer, error) {
	queue := make(chan []byte, queueSize)
	tp := &trackPlayer{
		queue:      queue,
		conn:       conn,
		stopSignal: stopSignal,
	}
	var err error
	tp.Player, err = audio.NewPlayer(func() pcm.FrameProvider {
		return tp.provider
	}, tp)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

func (tp *trackPlayer) ProvideAudio(mp3Data []byte) error {
	select {
	case tp.queue <- mp3Data:
		slog.Info("Enqueued new track for playback")
		return nil
	case <-tp.stopSignal:
		slog.Info("TrackPlayer closed, not enqueuing new track")
		return fmt.Errorf("track player is closed, cannot enqueue new track")
	default:
		slog.Warn("TrackPlayer queue is full, dropping new track")
		return fmt.Errorf("track player queue is full, cannot enqueue new track")
	}
}

// Suppose this is called when end of a track is reached
// Wait for the next track to be available in the queue
func (tp *trackPlayer) next() {
	select {
	case <-tp.stopSignal:
		slog.Info("TrackPlayer closed, stopping playback")
		return
	case audioData := <-tp.queue:
		provider, w, err := mp3.NewCustomPCMFrameProvider(nil, 48000, 1)
		if err != nil {
			slog.Error("Error creating mp3 provider", slog.Any("err", err))
			return
		}

		provider = pcm.NewPCMFrameChannelConverterProvider(provider, 48000, 1, 2)
		_, err = io.Copy(w, bytes.NewReader(audioData))
		if err != nil {
			slog.Error("Error writing to mp3 provider", slog.Any("err", err))
			return
		}

		tp.provider = provider
	}
}

func (p *trackPlayer) OnPause(player audio.Player) {}

func (p *trackPlayer) OnResume(player audio.Player) {}

func (p *trackPlayer) OnStart(player audio.Player) {}

func (p *trackPlayer) OnEnd(player audio.Player) {
	p.next()
}

func (p *trackPlayer) OnError(player audio.Player, err error) {
	slog.Error("Player error", slog.Any("err", err))
}

func (p *trackPlayer) OnClose(player audio.Player) {}
