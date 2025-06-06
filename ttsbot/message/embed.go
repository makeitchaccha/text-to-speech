package message

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/makeitchaccha/text-to-speech/ttsbot/i18n"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
)

var (
	colorDanger  = 0xed5555
	colorSuccess = 0x55ed55
	colorInfo    = 0x5555ed
)

func BuildPresetEmbed(preset preset.Preset, tr i18n.TextResource) *discord.EmbedBuilder {
	embedBuilder := discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Preset.Self).
		AddField(tr.Generic.Preset.Name, string(preset.Identifier), true).
		AddField(" ", " ", true). // dummy field for alignment
		AddField(tr.Generic.Preset.Language, preset.Language, true).
		AddField(tr.Generic.Preset.Engine, tr.Generic.Engines[preset.Engine], true).
		AddField(" ", " ", true). // dummy field for alignment
		AddField(tr.Generic.Preset.VoiceName, preset.VoiceName, true)

	if preset.SpeakingRate != 0 {
		embedBuilder.AddField("Speaking Rate", fmt.Sprintf("%.2f", preset.SpeakingRate), true)
	}

	return embedBuilder
}

func BuildJoinEmbed(tr i18n.TextResource, channelToRead, voiceChannel string) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.TTS.Ready).
		AddField(tr.Generic.TTS.ChannelToRead, channelToRead, true).
		AddField(tr.Generic.TTS.VoiceChannel, voiceChannel, true).
		SetColor(colorInfo)
}

func BuildLeaveEmbed(tr i18n.TextResource) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.TTS.End).
		SetDescription(tr.Generic.TTS.Thanks).
		SetColor(colorInfo)
}

func BuildSuccessEmbed(tr i18n.TextResource) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Success).
		SetColor(colorSuccess)
}

func BuildErrorEmbed(tr i18n.TextResource) *discord.EmbedBuilder {
	return discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Error).
		SetColor(colorDanger)
}

func BuildPresetListEmbed(presets []preset.Preset, tr i18n.TextResource) *discord.EmbedBuilder {
	embedBuilder := discord.NewEmbedBuilder().
		SetTitle(tr.Generic.Preset.List).
		SetDescription(fmt.Sprintf("### %s\n1. %s\n2. %s\n3. %s",
			tr.Generic.Preset.Name,
			tr.Generic.Preset.Engine,
			tr.Generic.Preset.Language,
			tr.Generic.Preset.VoiceName,
		)).
		SetColor(colorInfo)

	for _, p := range presets {
		embedBuilder.AddField(string(p.Identifier), fmt.Sprintf(
			"1. %s\n2. %s\n3. %s",
			tr.Generic.Engines[p.Engine],
			p.Language,
			p.VoiceName,
		), true)
	}

	return embedBuilder
}
