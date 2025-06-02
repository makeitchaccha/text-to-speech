package commands

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/makeitchaccha/text-to-speech/ttsbot/localization"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
	"github.com/makeitchaccha/text-to-speech/ttsbot/session"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

func joinCmd(trs localization.TextResources) discord.SlashCommandCreate {
	return discord.SlashCommandCreate{
		Name:        "join",
		Description: "Start text-to-speech in text channels",
		DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
			return tr.Commands.Join.Description
		}),
	}
}

func JoinHandler(engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, manager *session.Router) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {
		guildID := e.GuildID()

		if guildID == nil {
			return e.CreateMessage(discord.MessageCreate{
				Content: "This command can only be used in a guild.",
			})
		}

		// user must be in a voice channel to use this command
		voiceState, err := e.Client().Rest().GetUserVoiceState(*guildID, e.User().ID)
		var restErr rest.Error
		if ok := errors.As(err, &restErr); ok {
			switch restErr.Code {
			case 10065:
				return e.CreateMessage(discord.MessageCreate{
					Content: "You must be in a voice channel to use this command.",
				})
			case 50013:
				return e.CreateMessage(discord.MessageCreate{
					Content: "Bot does not have permission to view voice states in this guild.",
				})
			}
			slog.Warn("failed to get voice state", "error", err)
			return e.CreateMessage(discord.MessageCreate{
				Content: "failed to get voice state: " + err.Error(),
			})
		}

		if err != nil {
			slog.Warn("failed to get voice state", "error", err)
			return e.CreateMessage(discord.MessageCreate{
				Content: "failed to get voice state: " + err.Error(),
			})
		}

		if voiceState.ChannelID == nil {
			return e.CreateMessage(discord.MessageCreate{
				Content: "You must be in a voice channel to use this command.",
			})
		}

		voiceManager := e.Client().VoiceManager()
		conn := voiceManager.GetConn(*guildID)
		connected := conn != nil
		if connected && conn.ChannelID() == voiceState.ChannelID {
			return e.CreateMessage(discord.MessageCreate{
				Content: "Already connected to the voice channel.",
			})
		}

		if !connected {
			slog.Info("Creating voice connection", "guildID", *guildID, "channelID", voiceState.ChannelID)
			conn = voiceManager.CreateConn(*guildID)
		}

		err = e.DeferCreateMessage(false)
		if err != nil {
			return err
		}

		// Connect to the voice channel in go routine
		// Why? To establish the connection, we need to wait for the voice state update event
		// and waiting for it in the same goroutine would block the response from server.

		go func() {
			voiceChannelID := *voiceState.ChannelID

			slog.Info("Connecting to voice channel", "guildID", *guildID, "channelID", voiceChannelID)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := conn.Open(ctx, voiceChannelID, false, true); err != nil {
				slog.Warn("Failed to connect to voice channel", "error", err)
				e.UpdateInteractionResponse(discord.NewMessageUpdateBuilder().
					SetContent("Failed to connect to voice channel: " + err.Error()).Build(),
				)
				return
			}

			slog.Info("Connected to voice channel", "guildID", *guildID, "channelID", voiceChannelID)

			textChannel := e.Channel().ID()

			session, err := session.New(engineRegistry, presetResolver, textChannel, conn)
			if err != nil {
				slog.Error("Failed to create session", slog.Any("err", err), slog.String("textChannelID", textChannel.String()))
				e.UpdateInteractionResponse(discord.NewMessageUpdateBuilder().
					SetContent("Failed to create session: " + err.Error()).Build(),
				)
				conn.Close(context.Background())
				return
			}

			if _, err := e.UpdateInteractionResponse(discord.NewMessageUpdateBuilder().
				SetContentf("Connected to voice channel %s, %s", discord.ChannelMention(voiceChannelID), session).
				Build(),
			); err != nil {
				slog.Warn("Failed to update interaction response", "error", err)
			}

			slog.Info("Session created", "textChannelID", textChannel, "voiceChannelID", voiceChannelID)
			manager.Add(voiceChannelID, textChannel, session)
		}()

		return nil
	}
}
