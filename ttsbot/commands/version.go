package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/makeitchaccha/text-to-speech/ttsbot"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
)

func versionCmd(trs *i18n.TextResources) discord.SlashCommandCreate {
	return discord.SlashCommandCreate{
		Name:        "version",
		Description: "Show bot version information",
		DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
			return tr.Commands.Version.Description
		}),
	}
}

func VersionHandler(b *ttsbot.Bot) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(discord.NewEmbedBuilder().
				SetTitle("About Text-to-Speech Bot").
				SetDescription("Developed by **Make it! Chaccha**").
				AddField("Version", b.Version, true).
				AddField("Commit", b.Commit, true).
				Build(),
			).
			Build(),
		)
	}
}
