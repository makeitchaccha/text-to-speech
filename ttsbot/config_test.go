package ttsbot

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	configPath := filepath.Join("testdata", "config.toml")

	// Set environment variables to override values
	os.Setenv("TTSBOT_BOT_TOKEN", "env_bot_token")
	defer os.Unsetenv("TTSBOT_BOT_TOKEN")

	os.Setenv("TTSBOT_LOG_LEVEL", "warn")
	defer os.Unsetenv("TTSBOT_LOG_LEVEL")

	os.Setenv("TTSBOT_REDIS_URL", "redis://localhost:6379/2")
	defer os.Unsetenv("TTSBOT_REDIS_URL")

	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Assert values from config file, overridden by env vars where applicable
	assert.Equal(t, slog.LevelWarn, cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)
	assert.Equal(t, true, cfg.Log.AddSource)

	assert.Equal(t, []snowflake.ID{12345, 67890}, cfg.Bot.DevGuilds)
	assert.Equal(t, "env_bot_token", cfg.Bot.Token)
	assert.Equal(t, "en-US", cfg.Bot.Language)
	assert.Equal(t, "test-preset", cfg.Bot.FallbackPresetID)

	assert.Equal(t, "google", cfg.Presets["test-preset"].Engine)
	assert.Equal(t, "en-US", cfg.Presets["test-preset"].Language)
	assert.Equal(t, "en-US-Wavenet-A", cfg.Presets["test-preset"].VoiceName)
	assert.Equal(t, 1.0, cfg.Presets["test-preset"].SpeakingRate)

	assert.Equal(t, "sqlite3", cfg.Database.Driver)
	assert.Equal(t, "./test.db", cfg.Database.Dsn)

	assert.Equal(t, true, cfg.Redis.Enabled)
	assert.Equal(t, "redis://localhost:6379/2", cfg.Redis.Url)
	assert.Equal(t, 2*time.Hour, cfg.Redis.TTL)
}
