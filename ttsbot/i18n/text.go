package i18n

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
)

type TextResource struct {
	Generic struct {
		Guild   string `toml:"guild"`   // format: "guild"
		User    string `toml:"user"`    // format: "user"
		Success string `toml:"success"` // format: "Success"
		Error   string `toml:"error"`   // format: "Error"
		Preset  struct {
			Self         string `toml:"self"`          // format: "Preset"
			List         string `toml:"list"`          // format: "Preset List"
			Name         string `toml:"name"`          // format: "Preset Name"
			Engine       string `toml:"engine"`        // format: "Engine"
			Language     string `toml:"language"`      // format: "Language"
			VoiceName    string `toml:"voice_name"`    // format: "Voice Name"
			SpeakingRate string `toml:"speaking_rate"` // format: "Speaking Rate"
		} `toml:"preset"`
		TTS struct {
			Ready         string `toml:"ready"`           // format: "Text-to-Speech Ready"
			ChannelToRead string `toml:"channel_to_read"` // format: "Channel to Read"
			VoiceChannel  string `toml:"voice_channel"`   // format: "Voice Channel"
			End           string `toml:"end"`             // format: "Text-to-Speech Ended"
			Thanks        string `toml:"thanks"`          // format: "Thank you for using the Text-to-Speech service!"
		} `toml:"tts"`
	} `toml:"generic"`
	Commands struct {
		Generic struct {
			ErrorNotInGuild              string `toml:"error_not_in_guild"`             // format: "You must use this command in a guild"
			ErrorNotInVoiceChannel       string `toml:"error_not_in_voice_channel"`     // format: "You must be in a voice channel to use this command"
			ErrorInsufficientPermissions string `toml:"error_insufficient_permissions"` // format: "Bot has insufficient permissions."
		} `toml:"generic"`
		Join struct {
			Description         string `toml:"description"`           // format: "Start text-to-speech in text channels"
			ErrorAlreadyStarted string `toml:"error_already_started"` // format: "Text-to-speech has already been started"
		} `toml:"join"`
		Leave struct {
			Description     string `toml:"description"`       // format: "Stop text-to-speech in text channels"
			ErrorNotStarted string `toml:"error_not_started"` // format: "Text-to-speech is not started"
		} `toml:"leave"`
		Version struct {
			Description string `toml:"description"` // format: "Show bot version information"
		} `toml:"version"`
		Preset struct {
			Description string `toml:"description"` // format: "Manage presets for text-to-speech"
			Generic     struct {
				Description string `toml:"description"` // format: "Manage %[1]s presets"
				Set         struct {
					Description   string `toml:"description"`     // format: "Set a preset for the %[1]s"
					Name          string `toml:"name"`            // format: "Name of the preset to set"
					Success       string `toml:"success"`         // format: "Preset for %[1]s has been set to %[2]s"
					ErrorNotFound string `toml:"error_not_found"` // format: "Preset %[1]s not found"
					ErrorSave     string `toml:"error_save"`      // format: "Failed to save preset ID"
				} `toml:"set"`
				Unset struct {
					Description string `toml:"description"`  // format: "Unset a preset for the %[1]s"
					Success     string `toml:"success"`      // format: "Preset for %[1]s has been unset"
					ErrorDelete string `toml:"error_delete"` // format: "Failed to delete preset ID"
				} `toml:"unset"`
				Show struct {
					Description  string `toml:"description"`   // format: "Show the current preset for %[1]s"
					Current      string `toml:"current"`       // format: "Current preset for %[1]s"
					None         string `toml:"none"`          // format: "No preset set for %[1]s"
					ErrorFetch   string `toml:"error_fetch"`   // format: "Failed to fetch preset for %[1]s"
					ErrorInvalid string `toml:"error_invalid"` // format: "Preset ID is invalid. \nTo fix this, please set a new preset or unset the current preset."
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
		fallbackLocale:   discord.Locale(fallbackLocale),
	}

	if err := load(directory, resources.genericResources); err != nil {
		return nil, err
	}

	// validate that the fallback locale is present
	if _, ok := resources.genericResources[resources.fallbackLocale]; !ok {
		return nil, fmt.Errorf("fallback locale %s not found in text resources", fallbackLocale)
	}

	return resources, nil
}

// to make sure valid discord.Locale is used, we ignore LocaleUnknown
func (trs *TextResources) Localizations(f func(tr TextResource) string) map[discord.Locale]string {
	localizations := make(map[discord.Locale]string, len(trs.genericResources))
	for locale, resource := range trs.genericResources {
		if locale.String() == discord.LocaleUnknown.String() {
			continue
		}
		localizations[locale] = f(resource)
	}
	return localizations
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
