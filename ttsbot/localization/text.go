package localization

import (
	"github.com/disgoorg/disgo/discord"
)

type TextResources struct {
	genericResources[discord.Locale, TextResource]
}

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

func LoadTextResources(directory string) (*TextResources, error) {
	resources := &TextResources{
		genericResources: make(genericResources[discord.Locale, TextResource]),
	}

	if err := load(directory, resources.genericResources); err != nil {
		return nil, err
	}

	return resources, nil
}
