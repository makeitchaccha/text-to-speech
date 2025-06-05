package session

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"

	"github.com/disgoorg/audio"
	"github.com/disgoorg/audio/mp3"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/voice"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

type trackPlayer struct {
	audio.Player
	queue    <-chan *tts.SpeechResponse
	provider pcm.FrameProvider
	conn     voice.Conn
	close    <-chan struct{}
}

func newTrackPlayer(conn voice.Conn, queue <-chan *tts.SpeechResponse, close <-chan struct{}) (*trackPlayer, error) {
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
		provider, err := convertToFrameProvider(track)
		if err != nil {
			slog.Error("Failed to convert track to frame provider", slog.Any("error", err))
			return
		}
		p.provider = provider
	}
}

func convertToFrameProvider(resp *tts.SpeechResponse) (pcm.FrameProvider, error) {
	switch resp.Format {
	case tts.AudioFormatMp3:
		provider, w, err := mp3.NewCustomPCMFrameProvider(nil, 48000, 1)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, bytes.NewReader(resp.AudioContent)); err != nil {
			return nil, err
		}
		return pcm.NewPCMFrameChannelConverterProvider(provider, 48000, 1, 2), nil
	default:
		return nil, fmt.Errorf("unsupported audio format: %v", resp.Format)
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
