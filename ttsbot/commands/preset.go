package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
)

func presetCmd(trs *i18n.TextResources) discord.SlashCommandCreate {
	return discord.SlashCommandCreate{
		Name:        "preset",
		Description: "Manage presets for text-to-speech",
		DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
			return tr.Commands.Preset.Description
		}),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "guild",
				Description: "Manage guild presets",
				DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
					return fmt.Sprintf(tr.Commands.Preset.Generic.Description, tr.Generic.Guild)
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Set.Description, tr.Generic.Guild)
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
									return tr.Commands.Preset.Generic.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Unset.Description, tr.Generic.Guild)
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Show.Description, tr.Generic.Guild)
						}),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "user",
				Description: "Manage user presets",
				DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
					return fmt.Sprintf(tr.Commands.Preset.Generic.Description, tr.Generic.User)
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Set.Description, tr.Generic.User)
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
									return tr.Commands.Preset.Generic.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Unset.Description, tr.Generic.User)
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Show.Description, tr.Generic.User)
						}),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "list",
				Description: "List all presets",
				DescriptionLocalizations: trs.Localizations(func(tr i18n.TextResource) string {
					return tr.Commands.Preset.List.Description
				}),
			},
		},
	}
}

func PresetHandler(presetRegistry *preset.PresetRegistry, presetResolver preset.PresetResolver, presetIDRepository preset.PresetIDRepository, trs *i18n.TextResources) func(*handler.CommandEvent) error {
	return func(e *handler.CommandEvent) error {
		data := e.SlashCommandInteractionData()

		groupName := data.SubCommandGroupName
		if groupName != nil {
			return processPresetGroupCommand(e, presetRegistry, presetIDRepository, *groupName, trs)
		}

		return processPresetCommand(e, presetRegistry, trs)
	}
}

func processPresetGroupCommand(e *handler.CommandEvent, presetRegistry *preset.PresetRegistry, presetIDRepository preset.PresetIDRepository, groupName string, trs *i18n.TextResources) error {
	tr, ok := trs.Get(e.Locale())

	if !ok {
		slog.Error("failed to get localization for locale", "locale", e.Locale())
		tr = trs.GetFallback()
	}

	var scope preset.Scope
	var id snowflake.ID
	var generic string
	switch groupName {
	case "guild":
		scope = preset.ScopeGuild
		generic = tr.Generic.Guild
		id = *e.GuildID()
	case "user":
		scope = preset.ScopeUser
		generic = tr.Generic.User
		id = e.User().ID
	default:
		slog.Error("unknown preset group", "group", groupName)
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(message.BuildErrorEmbed(tr).
				SetDescription("Developer Error: Unsupported subcommand").
				Build()).
			Build())
	}

	data := e.SlashCommandInteractionData()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	switch *data.SubCommandName {
	case "set":
		preset, ok := presetRegistry.Get(preset.PresetID(data.String("name")))
		if !ok {
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescriptionf(tr.Commands.Preset.Generic.Set.ErrorNotFound, data.String("name")).
					Build()).
				Build())
		}

		err := presetIDRepository.Save(ctx, scope, id, preset.Identifier)
		if err != nil {
			slog.Error("failed to save preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescriptionf(tr.Commands.Preset.Generic.Set.ErrorSave, generic, err).
					Build()).
				Build())
		}

		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(message.BuildSuccessEmbed(tr).
				SetDescriptionf(tr.Commands.Preset.Generic.Set.Success, generic, preset.Identifier).
				Build(),
			).Build(),
		)

	case "unset":
		err := presetIDRepository.Delete(ctx, scope, id)
		if err != nil {
			slog.Error("failed to delete preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Preset.Generic.Unset.ErrorDelete).
					Build()).
				Build())
		}
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(message.BuildSuccessEmbed(tr).
				SetDescriptionf(tr.Commands.Preset.Generic.Unset.Success, generic).
				Build()).
			Build())

	case "show":
		presetID, err := presetIDRepository.Find(ctx, scope, id)
		if err != nil {
			if errors.Is(err, preset.ErrNotFound) {
				return e.CreateMessage(discord.NewMessageCreateBuilder().
					AddEmbeds(message.BuildErrorEmbed(tr).
						SetDescriptionf(tr.Commands.Preset.Generic.Show.None, generic).
						Build(),
					).
					Build())
			}
			slog.Error("failed to find preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Preset.Generic.Show.ErrorFetch).
					Build()).
				Build())
		}

		preset, ok := presetRegistry.Get(presetID)
		if !ok {
			slog.Error("failed to resolve preset", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				AddEmbeds(message.BuildErrorEmbed(tr).
					SetDescription(tr.Commands.Preset.Generic.Show.ErrorInvalid).
					Build()).
				Build())
		}
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			AddEmbeds(
				message.BuildPresetEmbed(preset, tr).
					SetDescriptionf(tr.Commands.Preset.Generic.Show.Current, generic).
					Build(),
			).
			Build())
	}

	return e.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("Developer Error: Unsupported subcommand").
		Build())
}

func processPresetCommand(e *handler.CommandEvent, presetRegistry *preset.PresetRegistry, trs *i18n.TextResources) error {
	data := e.SlashCommandInteractionData()
	tr, ok := trs.Get(e.Locale())
	if !ok {
		slog.Error("failed to get localization for locale", "locale", e.Locale())
		tr = trs.GetFallback()
	}

	switch *data.SubCommandName {
	case "list":
		presets := presetRegistry.List()

		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetEmbeds(message.BuildPresetListEmbed(presets, tr).Build()).
			Build())
	}

	slog.Error("unknown preset command", "command", *data.SubCommandName)
	return e.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("Developer Error: Unsupported subcommand").
		Build())
}
