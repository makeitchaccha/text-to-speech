package message

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/makeitchaccha/text-to-speech/ttsbot/localization"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
)

var (
	colorDanger  = 0xed5555
	colorSuccess = 0x55ed55
	colorInfo    = 0x5555ed
)

func BuildPresetEmbed(preset preset.Preset, tr localization.TextResource) *discord.EmbedBuilder {
	embedBuilder := discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Preset.Self).
		AddField(tr.Generic.Preset.Name, string(preset.Identifier), true).
		AddField(tr.Generic.Preset.Engine, preset.Engine, true).
		AddField(tr.Generic.Preset.Language, preset.Language, true).
		AddField(tr.Generic.Preset.VoiceName, preset.VoiceName, true)

	if preset.SpeakingRate != 0 {
		embedBuilder.AddField("Speaking Rate", fmt.Sprintf("%.2f", preset.SpeakingRate), true)
	}

	return embedBuilder
}

func BuildSuccessEmbed(tr localization.TextResource) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Success).
		SetColor(colorSuccess)
}

func BuildErrorEmbed(tr localization.TextResource) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Error).
		SetColor(colorDanger)
}

func BuildPresetListEmbed(presets []preset.Preset, tr localization.TextResource) *discord.EmbedBuilder {
	embedBuilder := discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Preset.List).
		SetColor(colorInfo)

	for _, p := range presets {
		embedBuilder.AddField(string(p.Identifier), fmt.Sprintf(
			"%s - %s\n%s - %s\n%s - %s",
			tr.Generic.Preset.Engine, p.Engine,
			tr.Generic.Preset.Language, p.Language,
			tr.Generic.Preset.VoiceName, p.VoiceName,
		), true)
	}

	return embedBuilder
}
