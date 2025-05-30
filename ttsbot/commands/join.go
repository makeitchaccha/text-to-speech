package commands

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/makeitchaccha/text-to-speech/ttsbot/session"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
	"github.com/samber/lo"
)

var join = discord.SlashCommandCreate{
	Name:        "join",
	Description: "Start text-to-speech in text channels",
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name:         "language",
			Description:  "Language for text-to-speech. If not provided, a default language will be used.",
			Required:     false,
			Autocomplete: true,
		},
		discord.ApplicationCommandOptionInt{
			Name:        "gender",
			Description: "gender of the voice for text-to-speech. If not provided, a system default voice will be used.",
			Choices: []discord.ApplicationCommandOptionChoiceInt{
				{
					Name:  "male",
					Value: int(texttospeechpb.SsmlVoiceGender_MALE),
				},
				{
					Name:  "female",
					Value: int(texttospeechpb.SsmlVoiceGender_FEMALE),
				},
				{
					Name:  "neutral",
					Value: int(texttospeechpb.SsmlVoiceGender_NEUTRAL),
				},
			},
			Required: false,
		},
		discord.ApplicationCommandOptionString{
			Name:         "voice",
			Description:  "Voice name for text-to-speech. If not provided, a system default voice will be used.",
			Required:     false,
			Autocomplete: true,
		},
	},
}

func JoinHandler(engine tts.Engine, manager *session.Router) handler.CommandHandler {
	return func(e *handler.CommandEvent) error {

		guildID := e.GuildID()

		if guildID == nil {
			return e.CreateMessage(discord.MessageCreate{
				Content: "This command can only be used in a guild.",
			})
		}

		// user must be in a voice channel to use this command
		voiceState, err := e.Client().Rest().GetUserVoiceState(*guildID, e.User().ID)
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
			if _, err := e.UpdateInteractionResponse(discord.NewMessageUpdateBuilder().
				SetContentf("Connected to voice channel %s", discord.ChannelMention(voiceChannelID)).
				Build(),
			); err != nil {
				slog.Warn("Failed to update interaction response", "error", err)
				conn.Close(context.Background())
				return
			}

			textChannel := e.Channel().ID()

			session := session.New(engine, textChannel, conn, session.WithLanguage("ja-JP"))
			manager.Add(voiceChannelID, textChannel, session)
		}()

		return nil
	}
}

func JoinAutocompleteHandler(ttsClient *texttospeech.Client) (func(e *handler.AutocompleteEvent) error, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := ttsClient.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		slog.Error("Failed to list voices", slog.Any("err", err))
		return nil, err
	}

	voiceNames := lo.Map(resp.Voices, func(voice *texttospeechpb.Voice, _ int) discord.AutocompleteChoice {
		return discord.AutocompleteChoiceString{
			Name:  voice.Name,
			Value: voice.Name,
		}
	})

	languages := lo.Map(lo.Uniq(lo.Flatten(lo.Map(resp.Voices, func(voice *texttospeechpb.Voice, _ int) []string {
		return voice.LanguageCodes
	}))), func(language string, _ int) discord.AutocompleteChoice {
		return discord.AutocompleteChoiceString{
			Name:  language,
			Value: language,
		}
	})

	slices.SortFunc(languages, func(a, b discord.AutocompleteChoice) int {
		return strings.Compare(a.ChoiceName(), b.ChoiceName())
	})

	return func(e *handler.AutocompleteEvent) error {
		focused := e.Data.Focused()
		switch focused.Name {
		case "language":
			language := strings.ToLower(e.Data.String("language"))
			languages := lo.Filter(languages, func(choice discord.AutocompleteChoice, _ int) bool {
				return strings.HasPrefix(strings.ToLower(choice.ChoiceName()), language)
			})

			if len(languages) > 25 {
				languages = languages[:25] // Discord limits autocomplete choices to 25
			}

			return e.AutocompleteResult(languages)

		case "voice":
			voice := strings.ToLower(e.Data.String("voice"))
			voices := lo.Filter(voiceNames, func(choice discord.AutocompleteChoice, _ int) bool {
				return strings.HasPrefix(strings.ToLower(choice.ChoiceName()), voice)
			})

			if len(voices) > 25 {
				voices = voices[:25] // Discord limits autocomplete choices to 25
			}

			return e.AutocompleteResult(voices)

		}

		return fmt.Errorf("unknown focused option: %s", focused.Name)
	}, nil
}
