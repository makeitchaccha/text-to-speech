package tts

import (
	"context"
	"encoding/hex"
	"hash"
	"hash/fnv"
	"log/slog"
	"time"

	"github.com/disgoorg/log"
	"github.com/go-redis/cache/v9"
)

var _ Engine = (*CachedTTSEngine)(nil)

// CachedTTSEngine is a wrapper around an Engine that caches the generated audio data.
// It uses redis to store the audio data with a key based on hash of the text, language code, and voice name.
type CachedTTSEngine struct {
	nextEngine Engine
	redisCache *cache.Cache
	ttl        time.Duration // Expiration time in seconds
	hash       hash.Hash
}

// NewCachedTTSEngine creates a new CachedTTSEngine with the provided nextEngine, redisCache, expiration time, and hash function.
func NewCachedTTSEngine(nextEngine Engine, redisCache *cache.Cache, ttl time.Duration, hash hash.Hash) *CachedTTSEngine {
	if hash == nil {
		hash = fnv.New64a()
	}

	return &CachedTTSEngine{
		nextEngine: nextEngine,
		redisCache: redisCache,
		hash:       hash,
	}
}

func (c *CachedTTSEngine) Name() string {
	return c.nextEngine.Name() + "-cached"
}

// Generate generates the audio data for the given text, language code, and voice name.
func (c *CachedTTSEngine) GenerateSpeech(ctx context.Context, request SpeechRequest) ([]byte, error) {
	key := c.generateKey(request)

	var audioData []byte
	err := c.redisCache.Get(ctx, key, &audioData)

	if err == nil {
		slog.Info("cache hit", "key", key, "engine", c.Name())
		return audioData, nil
	}

	audioData, err = c.nextEngine.GenerateSpeech(ctx, request)
	if err != nil {
		return nil, err
	}

	// Store the audio data in the cache with the generated key
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err = c.redisCache.Set(&cache.Item{
			Ctx:   ctx,
			Key:   key,
			Value: audioData,
			TTL:   c.ttl,
		}); err != nil {
			// Log the error but do not return it, as we don't want to fail the request if caching fails
			log.Warn("failed to cache audio data", "error", err, "key", key)
		}
	}()

	return audioData, nil
}

// generateKey creates a unique key for the cache based on the request parameters.
func (c *CachedTTSEngine) generateKey(request SpeechRequest) string {
	c.hash.Reset()
	c.hash.Write([]byte(c.nextEngine.Name()))
	c.hash.Write([]byte(request.LanguageCode))
	c.hash.Write([]byte(request.VoiceName))
	c.hash.Write([]byte(request.Text))
	return hex.EncodeToString(c.hash.Sum(nil))
}
