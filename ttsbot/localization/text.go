package localization

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/disgoorg/disgo/discord"
)

type TextResources map[discord.Locale]TextResource

type TextResource struct {
	Commands struct {
		Join struct {
			Description string `toml:"description"` // format: "Start text-to-speech in text channels"`
		} `toml:"join"`
		Version struct {
			Description string `toml:"description"` // format: "Show bot version information"`
		} `toml:"version"`
		Preset struct {
			Description string `toml:"description"` // format: "Manage presets for text-to-speech"`
			Guild       struct {
				Description string `toml:"description"` // format: "Manage guild presets"`
				Set         struct {
					Description string `toml:"description"` // format: "Set a preset for the guild"`
					Name        string `toml:"name"`        // format: "Name of the preset to set"`
				} `toml:"set"`
				Unset struct {
					Description string `toml:"description"` // format: "Unset a preset for the guild"`
				} `toml:"unset"`
				Show struct {
					Description string `toml:"description"` // format: "Show the current preset for the guild"`
				} `toml:"show"`
			} `toml:"guild"`
			User struct {
				Description string `toml:"description"` // format: "Manage user presets"`
				Set         struct {
					Description string `toml:"description"` // format: "Set a preset for the user"`
					Name        string `toml:"name"`        // format: "Name of the preset to set"`
				} `toml:"set"`
				Unset struct {
					Description string `toml:"description"` // format: "Unset a preset for the user"`
				} `toml:"unset"`
				Show struct {
					Description string `toml:"description"` // format: "Show the current preset for the user"`
				} `toml:"show"`
			} `toml:"user"`
			List struct {
				Description string `toml:"description"` // format: "List all presets"`
			} `toml:"list"`
		} `toml:"preset"`
	} `toml:"commands"`
}

func LoadTextResources(directory string) (TextResources, error) {
	trs := make(TextResources)
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read text resources directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip directories
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".toml") {
			// Skip non-TOML files
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".toml")

		filePath := path.Join(directory, entry.Name())

		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open text resource file %s: %w", filePath, err)
		}
		defer file.Close()

		var resource TextResource
		metadata, err := toml.NewDecoder(file).Decode(&resource)
		if err != nil {
			return nil, fmt.Errorf("failed to decode text resource file %s: %w", filePath, err)
		}

		if len(metadata.Undecoded()) > 0 {
			slog.Warn("text resource file contains undecoded fields", "file", filePath, "fields", metadata.Undecoded())
			return nil, fmt.Errorf("text resource file %s contains undecoded fields: %v", filePath, metadata.Undecoded())
		}

		discordLocale := discord.Locale(locale)
		if discordLocale.String() == discord.LocaleUnknown.String() {
			slog.Warn("text resource file has invalid locale", "file", filePath, "locale", locale)
			return nil, fmt.Errorf("text resource file %s has invalid locale: %s", filePath, locale)
		}
		trs[discordLocale] = resource
		slog.Info("Loaded text resource", "locale", locale, "file", filePath)
	}

	return trs, nil
}

func (tr TextResources) Localizations(value func(textResource TextResource) string) map[discord.Locale]string {
	localizations := make(map[discord.Locale]string)

	for locale, resource := range tr {
		localizations[locale] = value(resource)
	}

	return localizations
}

func (tr TextResources) Get(locale discord.Locale) TextResource {
	resource, ok := tr[locale]
	if !ok {
		return tr[discord.LocaleEnglishUS] // Fallback to English US if the requested locale is not found
	}
	return resource
}
