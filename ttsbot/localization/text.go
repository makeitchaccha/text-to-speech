package localization

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
)

type TextResource struct {
	Generic struct {
		Guild  string `toml:"guild"` // format: "guild"
		User   string `toml:"user"`  // format: "user"
		Preset struct {
			Self         string `toml:"self"`          // format: "Preset"
			Name         string `toml:"name"`          // format: "Preset Name"
			Engine       string `toml:"engine"`        // format: "Engine"
			Language     string `toml:"language"`      // format: "Language"
			VoiceName    string `toml:"voice_name"`    // format: "Voice Name"
			SpeakingRate string `toml:"speaking_rate"` // format: "Speaking Rate"
		} `toml:"preset"`
	} `toml:"generic"`
	Commands struct {
		Join struct {
			Description string `toml:"description"` // format: "Start text-to-speech in text channels"
		} `toml:"join"`
		Version struct {
			Description string `toml:"description"` // format: "Show bot version information"
		} `toml:"version"`
		Preset struct {
			Description string `toml:"description"` // format: "Manage presets for text-to-speech"
			Generic     struct {
				Description string `toml:"description"` // format: "Manage %[1]s presets"
				Set         struct {
					Description string `toml:"description"` // format: "Set a preset for the %[1]s"
					Name        string `toml:"name"`        // format: "Name of the preset to set"
				} `toml:"set"`
				Unset struct {
					Description string `toml:"description"` // format: "Unset a preset for the %[1]s"
				} `toml:"unset"`
				Show struct {
					Description string `toml:"description"` // format: "Show the current preset for %[1]s"
					Current     string `toml:"current"`     // format: "Current preset for %[1]s: %[2]s"
					None        string `toml:"none"`        // format: "No preset set for %[1]s"
				} `toml:"show"`
			} `toml:"generic"`
			List struct {
				Description string `toml:"description"` // format: "List all presets"
			} `toml:"list"`
		} `toml:"preset"`
	} `toml:"commands"`
}

type TextResources struct {
	genericResources[discord.Locale, TextResource]
	fallbackLocale discord.Locale
}

func LoadTextResources(directory string, fallbackLocale string) (*TextResources, error) {
	resources := &TextResources{
		genericResources: make(genericResources[discord.Locale, TextResource]),
	}

	if err := load(directory, resources.genericResources); err != nil {
		return nil, err
	}

	// validate that the fallback locale is present
	if _, ok := resources.genericResources[discord.Locale(fallbackLocale)]; !ok {
		return nil, fmt.Errorf("fallback locale %s not found in text resources", fallbackLocale)
	}

	return resources, nil
}

func (trs *TextResources) GetFallback() TextResource {
	resource, ok := trs.genericResources[trs.fallbackLocale]
	if !ok {
		// it won't happen because we validated it in LoadTextResources
		// but we panic here to make sure we catch it during development
		panic(fmt.Sprintf("fallback locale %s not found in text resources", trs.fallbackLocale))
	}
	return resource
}
