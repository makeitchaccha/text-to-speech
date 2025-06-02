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
	"github.com/makeitchaccha/text-to-speech/ttsbot/localization"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
)

func presetCmd(trs localization.TextResources) discord.SlashCommandCreate {
	return discord.SlashCommandCreate{
		Name:        "preset",
		Description: "Manage presets for text-to-speech",
		DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
			return tr.Commands.Preset.Description
		}),
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "guild",
				Description: "Manage guild presets",
				DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
					return tr.Commands.Preset.Guild.Description
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.Guild.Set.Description
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
									return tr.Commands.Preset.Guild.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.Guild.Unset.Description
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.Guild.Show.Description
						}),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "user",
				Description: "Manage user presets",
				DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
					return tr.Commands.Preset.User.Description
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.User.Set.Description
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
									return tr.Commands.Preset.User.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.User.Unset.Description
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return tr.Commands.Preset.User.Show.Description
						}),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "list",
				Description: "List all presets",
				DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
					return tr.Commands.Preset.List.Description
				}),
			},
		},
	}
}

func PresetHandler(presetRegistry *preset.PresetRegistry, presetResolver preset.PresetResolver, presetIDRepository preset.PresetIDRepository) func(*handler.CommandEvent) error {
	return func(e *handler.CommandEvent) error {
		data := e.SlashCommandInteractionData()

		groupName := data.SubCommandGroupName
		if groupName != nil {
			return processPresetGroupCommand(e, presetRegistry, presetResolver, presetIDRepository, *groupName)
		}

		return processPresetCommand(e, presetRegistry)
	}
}

func processPresetGroupCommand(e *handler.CommandEvent, presetRegistry *preset.PresetRegistry, presetResolver preset.PresetResolver, presetIDRepository preset.PresetIDRepository, groupName string) error {
	var scope preset.Scope
	var id snowflake.ID
	switch groupName {
	case "guild":
		scope = preset.ScopeGuild
		id = *e.GuildID()
	case "user":
		scope = preset.ScopeUser
		id = e.User().ID
	default:
		slog.Error("unknown preset group", "group", groupName)
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("Unknown preset group: " + groupName).
			Build())
	}

	data := e.SlashCommandInteractionData()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	switch *data.SubCommandName {
	case "set":

		err := presetIDRepository.Save(ctx, scope, id, preset.PresetID(data.String("name")))
		if err != nil {
			slog.Error("failed to save preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Failed to set preset for %s: %v", scope, err).
				Build())
		}
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("Preset for %s set to `%s`", scope, data.String("name")).
			Build())

	case "unset":
		err := presetIDRepository.Delete(ctx, scope, id)
		if err != nil {
			slog.Error("failed to delete preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Failed to unset preset for %s: %v", scope, err).
				Build())
		}
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("Preset for %s unset", scope).
			Build())
	case "show":
		presetID, err := presetIDRepository.Find(ctx, scope, id)
		if err != nil {
			if errors.Is(err, preset.ErrNotFound) {
				return e.CreateMessage(discord.NewMessageCreateBuilder().
					SetContentf("No preset set for %s", scope).
					Build())
			}
			slog.Error("failed to find preset ID", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Failed to show preset for %s: %v", scope, err).
				Build())
		}

		preset, ok := presetRegistry.Get(presetID)
		if !ok {
			slog.Error("failed to resolve preset", "error", err)
			return e.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Failed to resolve preset for %s: %v", scope, err).
				Build())
		}
		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("Current preset for %s: `%s` (Engine: `%s`, Language: `%s`, Voice: `%s`)",
				scope, presetID, preset.Engine, preset.Language, preset.VoiceName).
			Build())
	}

	return e.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("This command is not implemented yet.").
		Build())
}

func processPresetCommand(e *handler.CommandEvent, presetRegistry *preset.PresetRegistry) error {
	data := e.SlashCommandInteractionData()
	switch *data.SubCommandName {
	case "list":
		presets := presetRegistry.List()

		embedBuilder := discord.NewEmbedBuilder().
			SetTitle("Presets List")

		for _, preset := range presets {
			base := fmt.Sprintf("Engine: %s\nLanguage: %s\nVoice: %s\n", preset.Engine, preset.Language, preset.VoiceName)
			if preset.SpeakingRate != 0 {
				base += fmt.Sprintf("Speaking Rate: %.2f\n", preset.SpeakingRate)
			}
			embedBuilder.AddField(string(preset.Identifier), base, true)
		}

		return e.CreateMessage(discord.NewMessageCreateBuilder().
			SetEmbeds(embedBuilder.Build()).
			Build())
	}

	slog.Error("unknown preset command", "command", *data.SubCommandName)
	return e.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("This command is not implemented yet.").
		Build())
}
