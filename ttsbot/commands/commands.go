package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
)

func Commands(trs *i18n.TextResources) []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		joinCmd(trs),
		presetCmd(trs),
		versionCmd(trs),
	}
}
