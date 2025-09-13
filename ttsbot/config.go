package ttsbot

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")

	v.SetEnvPrefix("ttsbot")
	replacer := strings.NewReplacer(".", "_")
	v.SetEnvKeyReplacer(replacer)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if it's just not there
			fmt.Printf("Warning: Config file not found at %s, using defaults and environment variables.\n", path)
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	err := v.Unmarshal(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		stringToSlogLevelHookFunc(),
		stringToSnowflakeIDSliceHookFunc(),
	)))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

type Config struct {
	Log      LogConfig               `mapstructure:"log"`
	Bot      BotConfig               `mapstructure:"bot"`
	Presets  map[string]PresetConfig `mapstructure:"presets"`
	Database DatabaseConfig          `mapstructure:"database"`
	Redis    RedisConfig             `mapstructure:"redis"`
}

type BotConfig struct {
	DevGuilds        []snowflake.ID `mapstructure:"dev_guilds"`
	Token            string         `mapstructure:"token"`
	Language         string         `mapstructure:"default_lang"`
	FallbackPresetID string         `mapstructure:"fallback_preset_id"`
}

type LogConfig struct {
	Level     slog.Level `mapstructure:"level"`
	Format    string     `mapstructure:"format"`
	AddSource bool       `mapstructure:"add_source"`
}

type PresetConfig struct {
	Engine       string  `mapstructure:"engine"`
	Language     string  `mapstructure:"language"`
	VoiceName    string  `mapstructure:"voice_name"`
	SpeakingRate float64 `mapstructure:"speaking_rate"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	Dsn    string `mapstructure:"dsn"`
}

type RedisConfig struct {
	Enabled bool          `mapstructure:"enable"` // Note: changed from 'enabled' to 'enable' to match config.example.toml
	Url     string        `mapstructure:"url"`
	TTL     time.Duration `mapstructure:"ttl"`
}

func stringToSlogLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Kind,
		t reflect.Kind,
		data interface{},
	) (interface{}, error) {
		if f != reflect.String || t != reflect.TypeOf(slog.Level(0)).Kind() {
			return data, nil
		}
		// assert that data is a string
		switch s := data.(string); s {
		case "debug":
			return slog.LevelDebug, nil
		case "info":
			return slog.LevelInfo, nil
		case "warn":
			return slog.LevelWarn, nil
		case "error":
			return slog.LevelError, nil
		default:
			return nil, fmt.Errorf("unknown slog level: %s", s)
		}
	}
}

func stringToSnowflakeIDSliceHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Kind,
		t reflect.Kind,
		data interface{},
	) (interface{}, error) {
		if f != reflect.Slice || t != reflect.TypeOf([]snowflake.ID{}).Kind() {
			return data, nil
		}
		var ids []snowflake.ID
		for _, item := range data.([]interface{}) {
			str, ok := item.(string)
			if !ok {
				// If it's not a string, try to convert it to string (e.g., from int in TOML)
				str = fmt.Sprintf("%v", item)
			}
			id, err := snowflake.Parse(str)
			if err != nil {
				return nil, fmt.Errorf("failed to parse snowflake ID '%s': %w", str, err)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
}
