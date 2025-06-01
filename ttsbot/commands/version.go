package commands

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"

	"github.com/makeitchaccha/text-to-speech/ttsbot"
)

var version = discord.SlashCommandCreate{
	Name:        "version",
	Description: "version command",
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
