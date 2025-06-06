package session

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
)

type SessionManager interface {
	// GetByVoiceChannel retrieves a session by its voice channel ID.
	GetByVoiceChannel(voiceChannelID snowflake.ID) (*Session, bool)
	// GetByReadingChannel retrieves a session by its reading channel ID.
	GetByReadingChannel(readingChannelID snowflake.ID) (*Session, bool)
	// Add adds a new session with the given voice and reading channel IDs.
	Add(guildID, voiceChannelID, readingChannelID snowflake.ID, session *Session)
	// Delete removes a session by its voice channel ID.
	Delete(guildID, voiceChannelID snowflake.ID)

	// CreateMessageHandler creates an event listener for message creation events.
	CreateMessageHandler() bot.EventListener
	// CreateVoiceStateHandler creates an event listener for voice state update events.
	CreateVoiceStateHandler() bot.EventListener
	// GetByVoiceChannel retrieves a session by its voice channel ID.
}

var _ SessionManager = (*managerImpl)(nil)

type managerImpl struct {
	mu             sync.Mutex
	sessions       map[snowflake.ID]*Session
	readingToVoice map[snowflake.ID]snowflake.ID
	voiceToReading map[snowflake.ID]snowflake.ID

	persistenceManager PersistenceManager
}

func NewSessionManager(persistenceManger PersistenceManager) SessionManager {
	return &managerImpl{
		mu:                 sync.Mutex{},
		sessions:           make(map[snowflake.ID]*Session),
		readingToVoice:     make(map[snowflake.ID]snowflake.ID),
		voiceToReading:     make(map[snowflake.ID]snowflake.ID),
		persistenceManager: persistenceManger,
	}
}

func (r *managerImpl) GetByVoiceChannel(voiceChannelID snowflake.ID) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[voiceChannelID]
	return session, ok
}

func (r *managerImpl) GetByReadingChannel(readingChannelID snowflake.ID) (*Session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if voiceChannelID, ok := r.readingToVoice[readingChannelID]; ok {
		return r.sessions[voiceChannelID], true
	}
	return nil, false
}

func (r *managerImpl) Add(guildID, voiceChannelID, readingChannelID snowflake.ID, session *Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[voiceChannelID] = session
	r.readingToVoice[readingChannelID] = voiceChannelID
	r.voiceToReading[voiceChannelID] = readingChannelID
	if r.persistenceManager != nil {
		r.persistenceManager.Save(guildID, voiceChannelID, readingChannelID)
	}
}

func (r *managerImpl) Delete(guildID, voiceChannelID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, voiceChannelID)
	readingChannelID := r.voiceToReading[voiceChannelID]
	delete(r.readingToVoice, readingChannelID)
	delete(r.voiceToReading, voiceChannelID)
	if r.persistenceManager != nil {
		r.persistenceManager.Delete(guildID, voiceChannelID)
	}
}

func (m *managerImpl) CreateMessageHandler() bot.EventListener {
	return bot.NewListenerFunc(func(event *events.MessageCreate) {
		if session, ok := m.GetByReadingChannel(event.ChannelID); ok {
			session.onMessageCreate(event)
		}
	})
}

func (m *managerImpl) CreateVoiceStateHandler() bot.EventListener {
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

func (m *managerImpl) handleJoinVoiceChannel(event *events.GuildVoiceStateUpdate) {
	if session, ok := m.GetByVoiceChannel(*event.VoiceState.ChannelID); ok {
		session.onJoinVoiceChannel(event)
	}
}

func (m *managerImpl) handleLeaveVoiceChannel(event *events.GuildVoiceStateUpdate) {
	if session, ok := m.GetByVoiceChannel(*event.OldVoiceState.ChannelID); ok {
		result := session.onLeaveVoiceChannel(event)
		if result == LeaveResultClose {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			session.Close(ctx)
			m.Delete(event.OldVoiceState.GuildID, *event.OldVoiceState.ChannelID)
			_, err := event.Client().Rest().CreateMessage(session.textChannelID, discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildLeaveEmbed(*session.textResource).Build()).
				Build(),
			)
			if err != nil {
				event.Client().Logger().Error("Failed to send leave message", "error", err, "textChannelID", session.textChannelID)
			}
		}
	}
}
