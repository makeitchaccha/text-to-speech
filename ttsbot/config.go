package ttsbot

import (
	"fmt"
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
	if _, err = toml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	Log   LogConfig   `toml:"log"`
	Bot   BotConfig   `toml:"bot"`
	TTS   TTSConfig   `toml:"tts"`
	Redis RedisConfig `toml:"redis"`
}

type BotConfig struct {
	DevGuilds []snowflake.ID `toml:"dev_guilds"`
	Token     string         `toml:"token"`
	Language  string         `toml:"language"`
}

type LogConfig struct {
	Level     slog.Level `toml:"level"`
	Format    string     `toml:"format"`
	AddSource bool       `toml:"add_source"`
}

type TTSConfig struct {
	Language  string `toml:"language"`
	VoiceName string `toml:"voice_name"`
}

type RedisConfig struct {
	Enabled bool          `toml:"enabled"`
	Url     string        `toml:"url"`
	TTL     time.Duration `toml:"ttl"`
}
