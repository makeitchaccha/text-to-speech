package commands

import (
	"errors"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
	"github.com/makeitchaccha/text-to-speech/ttsbot/session"
)

func leaveCmd(trs *i18n.TextResources) discord.SlashCommandCreate {
	return discord.SlashCommandCreate{
		Name:        "leave",
		Description: "Stop text-to-speech in text channels",
		DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
			return tr.Commands.Leave.Description
		}),
	}
}

func LeaveHandler(manager *session.Router, trs *i18n.TextResources) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {
		tr, ok := trs.Get(e.Locale())
		if !ok {
			slog.Warn("text resource not found for locale", "locale", e.Locale())
			tr = trs.GetFallback()
		}

		voiceChannelID, err := SafeGetVoiceChannelID(e, tr)
		var friendlyErr *FriendlyError
		if ok := errors.As(err, &friendlyErr); ok {
			slog.Warn("Failed to get voice channel ID", "error", friendlyErr.err)
			return e.CreateMessage(friendlyErr.Message())
		}

		session, ok := manager.GetByVoiceChannel(*voiceChannelID)
		if !ok {
			slog.Warn("No active session found for voice channel", "channelID", *voiceChannelID)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Leave.ErrorNotStarted).
					Build()).
				Build())
		}

		// to prevent deadlock, close the session in a separate goroutine
		go func() {
			session.Close(e.Ctx)
			manager.Delete(*voiceChannelID)
		}()
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(message.BuildLeaveEmbed(tr).Build()).
			Build())

	}
}
