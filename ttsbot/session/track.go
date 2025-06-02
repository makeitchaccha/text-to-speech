package session

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/disgoorg/audio"
	"github.com/disgoorg/audio/mp3"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/voice"
)

type trackPlayer struct {
	audio.Player
	queue    <-chan []byte
	provider pcm.FrameProvider
	conn     voice.Conn
	close    <-chan struct{}
}

func newTrackPlayer(conn voice.Conn, queue <-chan []byte, close <-chan struct{}) (*trackPlayer, error) {
	player := &trackPlayer{
		queue: queue,
		conn:  conn,
		close: close,
	}
	var err error
	player.Player, err = audio.NewPlayer(func() pcm.FrameProvider {
		return player.provider
	}, player)
	if err != nil {
		return nil, err
	}
	return player, nil
}

func (p *trackPlayer) next() {
	select {
	case <-p.close:
		slog.Info("TrackPlayer closed, stopping playback")
		return
	case track := <-p.queue:
		provider, w, err := mp3.NewCustomPCMFrameProvider(nil, 48000, 1)
		if err != nil {
			slog.Error("Error creating mp3 provider", slog.Any("err", err))
		}
		p.provider = pcm.NewPCMFrameChannelConverterProvider(provider, 48000, 1, 2)
		_, _ = io.Copy(w, bytes.NewReader(track))
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
