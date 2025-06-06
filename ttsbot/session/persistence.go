package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/redis/go-redis/v9"
)

type (
	PersistenceManager interface {
		Save(guildID, voiceChannelID, readingChannelID snowflake.ID) error
		Delete(guildID, voiceChannelID snowflake.ID)
		StartHeartbeatLoop(interval time.Duration)

		// Restore restores a session from persistent store.
		Restore(ctx context.Context, sessionManager SessionManager, sessionRestoreFunc SessionRestoreFunc) error
	}

	SessionRestoreFunc func(guildID, voiceChannelID, readingChannelID snowflake.ID) (*Session, error)
)

type persistenceManagerImpl struct {
	redisClient        *redis.Client
	persistentSessions map[sessionID]persistentSession // guildID:voiceChannelID -> readingChannelID
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

func NewPersistenceManager(redisClient *redis.Client) PersistenceManager {
	return &persistenceManagerImpl{
		redisClient:        redisClient,
		persistentSessions: make(map[sessionID]persistentSession),
	}
}

func (p *persistenceManagerImpl) Save(guildID, voiceChannelID, readingChannelID snowflake.ID) error {
	key := sessionID{
		guildID:        guildID,
		voiceChannelID: voiceChannelID,
	}

	p.persistentSessions[key] = persistentSession{
		sessionID:        key,
		readingChannelID: readingChannelID,
	}
	return nil
}

func (p *persistenceManagerImpl) Delete(guildID, voiceChannelID snowflake.ID) {
	delete(p.persistentSessions, sessionID{
		guildID:        guildID,
		voiceChannelID: voiceChannelID,
	})
}

func (p *persistenceManagerImpl) StartHeartbeatLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	ttl := interval * 2 // Set TTL to twice the heartbeat interval
	go func() {
		for range ticker.C {
			for key, readingChannelID := range p.persistentSessions {
				sessionKey := key.generateKey()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := p.redisClient.Set(ctx, sessionKey, readingChannelID, ttl).Err(); err != nil {
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
			p.persistentSessions[session.sessionID] = session
			slog.Info("Restored session from Redis", "session", session)
		}
		cursor = nextCursor
	}

	return nil
}
