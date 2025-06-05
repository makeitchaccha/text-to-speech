package commands

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
)

var _ error = (*FriendlyError)(nil)

// command-related utilities
type FriendlyError struct {
	err     error
	message discord.MessageCreate
}

func newFriendlyError(err error, message discord.MessageCreate) *FriendlyError {
	return &FriendlyError{
		err:     err,
		message: message,
	}
}
func (e FriendlyError) Error() string {
	return e.err.Error()
}

func (e FriendlyError) Message() discord.MessageCreate {
	return e.message
}

func SafeGetVoiceChannelID(e *handler.CommandEvent, tr i18n.TextResource) (*snowflake.ID, error) {
	if e.Context() != discord.InteractionContextTypeGuild {
		return nil, newFriendlyError(
			fmt.Errorf("command cannot be used outside of a guild"),
			discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Generic.ErrorNotInGuild).
					Build()).
				Build(),
		)
	}

	guildID := e.GuildID()

	// user must be in a voice channel to use this command
	voiceState, err := e.Client().Rest().GetUserVoiceState(*guildID, e.User().ID)
	var restErr rest.Error
	if ok := errors.As(err, &restErr); ok {
		switch restErr.Code {
		case 10065:
			return nil, newFriendlyError(
				fmt.Errorf("user not in a voice channel"),
				discord.NewMessageCreateBuilder().
					AddEmbeds(message.BuildErrorEmbed(tr).
						SetDescription(tr.Commands.Generic.ErrorNotInVoiceChannel).
						Build()).
					Build(),
			)
		case 50013:
			return nil, newFriendlyError(
				fmt.Errorf("missing permissions to get voice state"),
				discord.NewMessageCreateBuilder().
					AddEmbeds(message.BuildErrorEmbed(tr).
						SetDescription(tr.Commands.Generic.ErrorInsufficientPermissions).
						Build()).
					Build(),
			)
		}
	}

	if err != nil {
		slog.Warn("failed to get voice state", "error", err)
		return nil, newFriendlyError(
			fmt.Errorf("failed to get voice state: %w", err),
			discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription("Failed to get voice state").
					Build()).
				Build(),
		)
	}

	if voiceState.ChannelID == nil {
		return nil, newFriendlyError(
			fmt.Errorf("user not in a voice channel"),
			discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Generic.ErrorNotInVoiceChannel).
					Build()).
				Build(),
		)
	}

	return voiceState.ChannelID, nil
}
