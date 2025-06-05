package i18n

import (
	"github.com/disgoorg/disgo/discord"
)

type VoiceResources struct {
	genericResources[discord.Locale, VoiceResource]
}

type VoiceResource struct {
	Session struct {
		Launch      string `toml:"launch"`      // "Ready to start text-to-speech in this channel."
		UserJoin    string `toml:"user_join"`   // "%[1]s has joined the voice channel."
		UserLeave   string `toml:"user_leave"`  // "%[1]s has left the voice channel."
		Attachments string `toml:"attachments"` // "%[1]d attachments"
	} `toml:"session"`
}

func LoadVoiceResources(directory string) (*VoiceResources, error) {
	resources := &VoiceResources{
		genericResources: make(genericResources[discord.Locale, VoiceResource]),
	}

	if err := load(directory, resources.genericResources); err != nil {
		return nil, err
	}

	return resources, nil
}
