package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/makeitchaccha/text-to-speech/ttsbot/localization"
)

func Commands(trs localization.TextResources) []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		joinCmd(trs),
		presetCmd(trs),
		versionCmd(trs),
	}
}
