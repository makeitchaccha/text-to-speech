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

func presetCmd(trs *localization.TextResources) discord.SlashCommandCreate {
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
					return fmt.Sprintf(tr.Commands.Preset.Generic.Description, tr.Generic.Guild)
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Set.Description, tr.Generic.Guild)
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
									return tr.Commands.Preset.Generic.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Unset.Description, tr.Generic.Guild)
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the guild",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Show.Description, tr.Generic.Guild)
						}),
					},
				},
			},
			discord.ApplicationCommandOptionSubCommandGroup{
				Name:        "user",
				Description: "Manage user presets",
				DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
					return fmt.Sprintf(tr.Commands.Preset.Generic.Description, tr.Generic.User)
				}),
				Options: []discord.ApplicationCommandOptionSubCommand{
					{
						Name:        "set",
						Description: "Set a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Set.Description, tr.Generic.User)
						}),
						Options: []discord.ApplicationCommandOption{
							discord.ApplicationCommandOptionString{
								Name:        "name",
								Description: "Name of the preset to set",
								DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
									return tr.Commands.Preset.Generic.Set.Name
								}),
							},
						},
					},
					{
						Name:        "unset",
						Description: "Unset a preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Unset.Description, tr.Generic.User)
						}),
					},
					{
						Name:        "show",
						Description: "Show the current preset for the user",
						DescriptionLocalizations: trs.Localizations(func(tr localization.TextResource) string {
							return fmt.Sprintf(tr.Commands.Preset.Generic.Show.Description, tr.Generic.User)
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

func PresetHandler(presetRegistry *preset.PresetRegistry, presetResolver preset.PresetResolver, presetIDRepository preset.PresetIDRepository, trs *localization.TextResources) func(*handler.CommandEvent) error {
	return func(e *handler.CommandEvent) error {
		data := e.SlashCommandInteractionData()

		groupName := data.SubCommandGroupName
		if groupName != nil {
			return processPresetGroupCommand(e, presetRegistry, presetIDRepository, *groupName, trs)
		}

		return processPresetCommand(e, presetRegistry)
	}
}

func processPresetGroupCommand(e *handler.CommandEvent, presetRegistry *preset.PresetRegistry, presetIDRepository preset.PresetIDRepository, groupName string, trs *localization.TextResources) error {
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
			AddEmbeds(discord.NewEmbedBuilder().
				SetTitle("Error").
				SetDescription("Developer Error: Unsupported subcommand").
				SetColor(0xed5555).
				Build()).
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
			SetContentf(tr.Commands.Preset.Generic.Show.None, generic).
			Build())
	case "show":
		presetID, err := presetIDRepository.Find(ctx, scope, id)
		if err != nil {
			if errors.Is(err, preset.ErrNotFound) {
				return e.CreateMessage(discord.NewMessageCreateBuilder().
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
			AddEmbeds(buildPresetEmbed(preset, tr)).
			Build())
	}

	return e.CreateMessage(discord.NewMessageCreateBuilder().
		SetContent("This command is not implemented yet.").
		Build())
}

func buildPresetEmbed(preset preset.Preset, trs localization.TextResource) discord.Embed {
	embedBuilder := discord.NewEmbedBuilder().
		SetTitle(trs.Generic.Preset.Self).
		AddField(trs.Generic.Preset.Name, string(preset.Identifier), true).
		AddField(trs.Generic.Preset.Engine, preset.Engine, true).
		AddField(trs.Generic.Preset.Language, preset.Language, true).
		AddField(trs.Generic.Preset.VoiceName, preset.VoiceName, true)

	if preset.SpeakingRate != 0 {
		embedBuilder.AddField("Speaking Rate", fmt.Sprintf("%.2f", preset.SpeakingRate), true)
	}

	return embedBuilder.Build()
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
