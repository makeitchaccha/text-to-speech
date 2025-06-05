package ttsbot

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/disgoorg/snowflake/v2"
)

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}

	var cfg Config
	md, err := toml.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, err
	}

	if len(md.Undecoded()) > 0 {
		return nil, fmt.Errorf("config contains undecoded fields: %v", md.Undecoded())
	}

	overrideString(&cfg.Bot.Token, "BOT_TOKEN")

	return &cfg, nil
}

func overrideString(value *string, envVar string) {
	if envValue, exists := os.LookupEnv(envVar); exists && envValue != "" {
		*value = envValue
	} else if *value == "" {
		log.Printf("Warning: %s is not set in the config or environment, using empty string as default", envVar)
	}
}

type Config struct {
	Log      LogConfig               `toml:"log"`
	Bot      BotConfig               `toml:"bot"`
	Presets  map[string]PresetConfig `toml:"presets"`
	Database DatabaseConfig          `toml:"database"`
	Redis    RedisConfig             `toml:"redis"`
}

type BotConfig struct {
	DevGuilds        []snowflake.ID `toml:"dev_guilds"`
	Token            string         `toml:"token"`
	Language         string         `toml:"language"`
	FallbackPresetID string         `toml:"fallback_preset_id"`
}

type LogConfig struct {
	Level     slog.Level `toml:"level"`
	Format    string     `toml:"format"`
	AddSource bool       `toml:"add_source"`
}

type PresetConfig struct {
	Engine       string  `toml:"engine"`
	Language     string  `toml:"language"`
	VoiceName    string  `toml:"voice_name"`
	SpeakingRate float64 `toml:"speaking_rate"`
}

type DatabaseConfig struct {
	Driver string `toml:"driver"`
	Dsn    string `toml:"dsn"`
}

type RedisConfig struct {
	Enabled bool          `toml:"enabled"`
	Url     string        `toml:"url"`
	TTL     time.Duration `toml:"ttl"`
}
