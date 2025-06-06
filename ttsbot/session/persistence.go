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

type (
	PersistenceManager interface {
		Save(guildID, voiceChannelID, readingChannelID snowflake.ID)
		Delete(guildID, voiceChannelID snowflake.ID)
		StartHeartbeatLoop()

		// Restore restores a session from persistent store.
		Restore(ctx context.Context, sessionManager SessionManager, sessionRestoreFunc SessionRestoreFunc) error
	}

	SessionRestoreFunc func(guildID, voiceChannelID, readingChannelID snowflake.ID) (*Session, error)
)

type persistenceManagerImpl struct {
	redisClient        *redis.Client
	persistentSessions map[sessionID]persistentSession // guildID:voiceChannelID -> readingChannelID
	heatbeatInterval   time.Duration
}

const (
	keySessionPrefix = "session"
)

type sessionID struct {
	guildID        snowflake.ID
	voiceChannelID snowflake.ID
}

func (s sessionID) generateKey() string {
	return fmt.Sprintf(keySessionPrefix+":%d:%d", s.guildID, s.voiceChannelID)
}

type persistentSession struct {
	sessionID
	readingChannelID snowflake.ID
}

var _ encoding.BinaryMarshaler = (*persistentSession)(nil)
var _ encoding.BinaryUnmarshaler = (*persistentSession)(nil)

func (s *persistentSession) MarshalBinary() ([]byte, error) {
	// marshal with binary encoding
	data := make([]byte, 8+8+8) // 3 snowflake IDs, each 8 bytes
	binary.BigEndian.PutUint64(data[0:8], uint64(s.guildID))
	binary.BigEndian.PutUint64(data[8:16], uint64(s.voiceChannelID))
	binary.BigEndian.PutUint64(data[16:24], uint64(s.readingChannelID))
	return data, nil
}

func (s *persistentSession) UnmarshalBinary(data []byte) error {
	if len(data) != 24 {
		return fmt.Errorf("invalid data length: expected 24 bytes, got %d", len(data))
	}
	s.guildID = snowflake.ID(binary.BigEndian.Uint64(data[0:8]))
	s.voiceChannelID = snowflake.ID(binary.BigEndian.Uint64(data[8:16]))
	s.readingChannelID = snowflake.ID(binary.BigEndian.Uint64(data[16:24]))
	return nil
}

func NewPersistenceManager(redisClient *redis.Client, heatbeatInterval time.Duration) PersistenceManager {
	return &persistenceManagerImpl{
		redisClient:        redisClient,
		persistentSessions: make(map[sessionID]persistentSession),
		heatbeatInterval:   heatbeatInterval,
	}
}

func (p *persistenceManagerImpl) Save(guildID, voiceChannelID, readingChannelID snowflake.ID) {
	key := sessionID{
		guildID:        guildID,
		voiceChannelID: voiceChannelID,
	}

	session := persistentSession{
		sessionID:        key,
		readingChannelID: readingChannelID,
	}
	p.persistentSessions[key] = session

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.redisClient.Set(ctx, key.generateKey(), &session, p.ttl()).Err(); err != nil {
			slog.Error("Failed to persist session to Redis", slog.Any("sessionKey", key), slog.Any("error", err))
		}
	}()
}

func (p *persistenceManagerImpl) Delete(guildID, voiceChannelID snowflake.ID) {
	delete(p.persistentSessions, sessionID{
		guildID:        guildID,
		voiceChannelID: voiceChannelID,
	})
}

func (p *persistenceManagerImpl) StartHeartbeatLoop() {
	ticker := time.NewTicker(p.heatbeatInterval)
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

func (p *persistenceManagerImpl) Restore(ctx context.Context, sessionManager SessionManager, sessionRestoreFunc SessionRestoreFunc) error {
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

func (p *persistenceManagerImpl) ttl() time.Duration {
	return p.heatbeatInterval * 3
}
