package session

import (
	"context"
	"encoding"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/redis/go-redis/v9"
)

type SessionRestoreFunc func(guildID, voiceChannelID, readingChannelID snowflake.ID) (*Session, error)

var _ SessionLifecycleObserver = (*PersistenceManager)(nil)

type PersistenceManager struct {
	NoOpSessionLifecycleObserver

	// applicationID for the persistence manager in the redis store.
	// If multiple instances of the bot are running, they should have different identifiers.
	// recommended to use the bot's application ID but it can be any unique.
	applicationID      snowflake.ID
	redisClient        *redis.Client
	persistentSessions map[sessionID]persistentSession // guildID:voiceChannelID -> readingChannelID
	heartbeatInterval  time.Duration
}

const (
	keySessionPrefix = "session"
)

type sessionID struct {
	applicationID  snowflake.ID
	voiceChannelID snowflake.ID
}

func (s sessionID) generateKey() string {
	return fmt.Sprintf(keySessionPrefix+":%d:%d", s.applicationID, s.voiceChannelID)
}

type persistentSession struct {
	applicationID    snowflake.ID
	guildID          snowflake.ID
	voiceChannelID   snowflake.ID
	readingChannelID snowflake.ID
}

var _ encoding.BinaryMarshaler = (*persistentSession)(nil)
var _ encoding.BinaryUnmarshaler = (*persistentSession)(nil)

func (s *persistentSession) MarshalBinary() ([]byte, error) {
	// marshal with binary encoding
	data := make([]byte, 8+8+8+8)
	binary.BigEndian.PutUint64(data[0:8], uint64(s.applicationID))
	binary.BigEndian.PutUint64(data[8:16], uint64(s.guildID))
	binary.BigEndian.PutUint64(data[16:24], uint64(s.voiceChannelID))
	binary.BigEndian.PutUint64(data[24:32], uint64(s.readingChannelID))
	return data, nil
}

func (s *persistentSession) UnmarshalBinary(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid data length: expected 32 bytes, got %d", len(data))
	}

	s.applicationID = snowflake.ID(binary.BigEndian.Uint64(data[0:8]))
	s.guildID = snowflake.ID(binary.BigEndian.Uint64(data[8:16]))
	s.voiceChannelID = snowflake.ID(binary.BigEndian.Uint64(data[16:24]))
	s.readingChannelID = snowflake.ID(binary.BigEndian.Uint64(data[24:32]))
	return nil
}

func NewPersistenceManager(applicationID snowflake.ID, redisClient *redis.Client, heatbeatInterval time.Duration) *PersistenceManager {
	return &PersistenceManager{
		redisClient:        redisClient,
		applicationID:      applicationID,
		persistentSessions: make(map[sessionID]persistentSession),
		heartbeatInterval:  heatbeatInterval,
	}
}

func (p *PersistenceManager) OnCreated(e SessionCreatedEvent) {
	key := sessionID{
		applicationID:  p.applicationID,
		voiceChannelID: e.VoiceChannelID,
	}

	session := persistentSession{
		applicationID:    p.applicationID,
		guildID:          e.GuildID,
		voiceChannelID:   e.VoiceChannelID,
		readingChannelID: e.ReadingChannelID,
	}
	p.persistentSessions[key] = session

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.redisClient.Set(ctx, key.generateKey(), &session, p.ttl()).Err(); err != nil {
		slog.Error("Failed to persist session to Redis", slog.Any("sessionKey", key), slog.Any("error", err))
	}
}

func (p *PersistenceManager) OnDeleted(e SessionDeletedEvent) {
	delete(p.persistentSessions, sessionID{
		applicationID:  p.applicationID,
		voiceChannelID: e.VoiceChannelID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.redisClient.Del(ctx, sessionID{
		applicationID:  p.applicationID,
		voiceChannelID: e.VoiceChannelID,
	}.generateKey()).Err(); err != nil {
		slog.Error("Failed to delete session from Redis", slog.Any("sessionKey", e.VoiceChannelID), slog.Any("error", err))
	}
	slog.Debug("Deleted session from Redis", slog.Any("voiceChannelID", e.VoiceChannelID))
}

func (p *PersistenceManager) StartHeartbeatLoop() {
	ticker := time.NewTicker(p.heartbeatInterval)
	ttl := p.ttl()
	go func() {
		for range ticker.C {
			for key, session := range p.persistentSessions {
				sessionKey := key.generateKey()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := p.redisClient.Set(ctx, sessionKey, &session, ttl).Err(); err != nil {
					slog.Error("Failed to persist session to Redis", slog.Any("sessionKey", sessionKey), slog.Any("error", err))
					cancel()
					continue
				}
				cancel()
			}
			slog.Debug("Persisted sessions to Redis")
		}
	}()
}

func (p *PersistenceManager) Restore(ctx context.Context, sessionManager SessionManager, sessionRestoreFunc SessionRestoreFunc) error {
	for done, cursor := false, uint64(0); !done; done = cursor == 0 {
		keys, nextCursor, err := p.redisClient.Scan(ctx, cursor, keySessionPrefix+":*", 100).Result()
		if err != nil {
			slog.Error("Failed to scan Redis for sessions", slog.Any("error", err))
			return fmt.Errorf("failed to scan Redis for sessions: %w", err)
		}

		if len(keys) == 0 {
			slog.Debug("No sessions found in Redis")
			return nil
		}
		for _, key := range keys {
			var session persistentSession
			err = p.redisClient.Get(ctx, key).Scan(&session)
			if err != nil {
				slog.Warn("Failed to get session from Redis", slog.Any("key", key), slog.Any("error", err))
				// just ignore this session if it cannot be retrieved
				continue
			}

			if session.applicationID != p.applicationID {
				slog.Debug("Skipping session from different application ID", slog.Any("session", session), slog.Any("applicationID", p.applicationID))
				// skip sessions that are not from this application ID
				continue
			}

			// conn.Open() blocks until the voice state update event is received...
			// so we need to restore the session in a separate goroutine
			go func() {
				s, err := sessionRestoreFunc(session.guildID, session.voiceChannelID, session.readingChannelID)
				if err != nil {
					slog.Error("Failed to restore session", slog.Any("session", session), slog.Any("error", err))
					return
				}
				sessionManager.Add(session.guildID, session.voiceChannelID, session.readingChannelID, s)
				slog.Info("Restored session from Redis", "session", session)
			}()
		}
		cursor = nextCursor
	}

	return nil
}

func (p *PersistenceManager) ttl() time.Duration {
	return p.heartbeatInterval * 3
}
