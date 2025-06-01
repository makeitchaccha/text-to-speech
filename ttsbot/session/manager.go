package session

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

type Router struct {
	mu             sync.Mutex
	sessions       map[snowflake.ID]*Session
	readingToVoice map[snowflake.ID]snowflake.ID
	voiceToReading map[snowflake.ID]snowflake.ID
}

func NewRouter() *Router {
	return &Router{
		mu:             sync.Mutex{},
		sessions:       make(map[snowflake.ID]*Session),
		readingToVoice: make(map[snowflake.ID]snowflake.ID),
		voiceToReading: make(map[snowflake.ID]snowflake.ID),
	}
}

func (r *Router) GetByVoiceChannel(voiceChannelID snowflake.ID) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[voiceChannelID]
	return session, ok
}

func (r *Router) GetByReadingChannel(readingChannelID snowflake.ID) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if voiceChannelID, ok := r.readingToVoice[readingChannelID]; ok {
		return r.sessions[voiceChannelID], true
	}
	return nil, false
}

func (r *Router) Add(voiceChannelID snowflake.ID, readingChannelID snowflake.ID, session *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[voiceChannelID] = session
	r.readingToVoice[readingChannelID] = voiceChannelID
	r.voiceToReading[voiceChannelID] = readingChannelID
}

func (r *Router) Delete(channelID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, channelID)
	readingChannelID := r.voiceToReading[channelID]
	delete(r.readingToVoice, readingChannelID)
	delete(r.voiceToReading, channelID)
}

func (m *Router) CreateMessageHandler() bot.EventListener {
	return bot.NewListenerFunc(func(event *events.MessageCreate) {
		if session, ok := m.GetByReadingChannel(event.ChannelID); ok {
			session.onMessageCreate(event)
		}
	})
}

func (m *Router) CreateVoiceStateHandler() bot.EventListener {
	return bot.NewListenerFunc(func(event *events.GuildVoiceStateUpdate) {
		if event.OldVoiceState.ChannelID == nil {
			m.handleJoinVoiceChannel(event)
			return
		}

		if event.VoiceState.ChannelID == nil {
			m.handleLeaveVoiceChannel(event)
			return
		}

		if *event.OldVoiceState.ChannelID != *event.VoiceState.ChannelID {
			m.handleLeaveVoiceChannel(event)
			m.handleJoinVoiceChannel(event)
		}

	})
}

func (m *Router) handleJoinVoiceChannel(event *events.GuildVoiceStateUpdate) {
	if session, ok := m.GetByVoiceChannel(*event.VoiceState.ChannelID); ok {
		session.onJoinVoiceChannel(event)
	}
}

func (m *Router) handleLeaveVoiceChannel(event *events.GuildVoiceStateUpdate) {
	if session, ok := m.GetByVoiceChannel(*event.OldVoiceState.ChannelID); ok {
		result := session.onLeaveVoiceChannel(event)
		if result == LeaveResultClose {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			session.Close(ctx)
			m.Delete(*event.OldVoiceState.ChannelID)
		}
	}
}
