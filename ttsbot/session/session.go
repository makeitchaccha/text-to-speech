package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"github.com/makeitchaccha/text-to-speech/ttsbot/audio"
	"github.com/makeitchaccha/text-to-speech/ttsbot/localization"
	"github.com/makeitchaccha/text-to-speech/ttsbot/message"
	"github.com/makeitchaccha/text-to-speech/ttsbot/preset"
	"github.com/makeitchaccha/text-to-speech/ttsbot/tts"
)

type LeaveResult int

// LeaveResult indicates which action to take after a user leaves the voice channel.
// LeaveResultKeepAlive means to keep the session alive, allowing it to continue processing messages.
// LeaveResultClose means to close the session, as there are no users left in the voice channel.
const (
	LeaveResultKeepAlive LeaveResult = iota
	LeaveResultClose
)

type Session struct {
	engineRegistry *tts.EngineRegistry
	presetResolver preset.PresetResolver
	guildID        snowflake.ID
	textChannelID  snowflake.ID
	worker         audio.AudioWorker
	voiceResources *localization.VoiceResources
}

func New(engineRegistry *tts.EngineRegistry, presetResolver preset.PresetResolver, guildID, textChannelID snowflake.ID, worker audio.AudioWorker, vrs *localization.VoiceResources) (*Session, error) {
	session := &Session{
		engineRegistry: engineRegistry,
		presetResolver: presetResolver,
		guildID:        guildID,
		textChannelID:  textChannelID,
		worker:         worker,
		voiceResources: vrs,
	}

	go session.worker.Start()

	return session, nil
}

func (s *Session) AnnounceReady(ctx context.Context) {
	currentPreset, err := s.presetResolver.ResolveGuildPreset(ctx, s.guildID)
	if err != nil {
		slog.Error("Failed to resolve preset for session announcement", slog.Any("err", err), slog.String("guildID", s.guildID.String()))
		return
	}

	vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(currentPreset.Language))
	if !ok {
		slog.Warn("Voice resources not found for locale for session announcement", "locale", currentPreset.Language, "guildID", s.guildID.String())
		return
	}
	s.worker.EnqueueTask(audio.NewSpeechTask(currentPreset, []string{
		vr.Session.Launch,
	}))
	slog.Info("Enqueued session ready announcement", "guildID", s.guildID, "textChannelID", s.textChannelID)
}

func (s *Session) Close(ctx context.Context) {
	s.worker.Stop()
}

func (s *Session) onMessageCreate(event *events.MessageCreate) {
	// ignore messages from other channels or from bots
	if event.Message.Author.Bot {
		return
	}

	slog.Debug("Received message for TTS", "messageID", event.Message.ID, "content", event.Message.Content)

	member, err := event.Client().Rest().GetMember(*event.GuildID, event.Message.Author.ID)
	if err != nil {
		slog.Error("Failed to get member for message author", slog.Any("err", err), slog.String("userID", event.Message.Author.ID.String()))
		return
	}

	mentions := createIdToNameMap(event.Client(), *event.GuildID, event.Message.Mentions)

	// make the content safe and ready for TTS.
	content := event.Message.Content
	content = message.ReplaceUserMentions(content, mentions)
	content = message.ReplaceUrlsWithPlaceholders(content)
	content = message.ConvertMarkdownToPlainText(content)
	content = message.LimitContentLength(content, 300)

	segments := make([]string, 0)
	segments = append(segments, content)
	if nAttachment := len(event.Message.Attachments); nAttachment > 0 {
		segments = append(segments, fmt.Sprintf("%d attachments", nAttachment))
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		preset, err := s.presetResolver.Resolve(ctx, *event.GuildID, event.Message.Author.ID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err), slog.String("content", content))
			return
		}

		s.worker.EnqueueTask(audio.NewSpeechTask(preset, segments, audio.WithSpeaker(event.Message.Author.ID, member.EffectiveName())))
		slog.Info("Enqueued speech task", "content", content, "preset", preset.Identifier)
	}()
}

func createIdToNameMap(client bot.Client, guildID snowflake.ID, users []discord.User) map[snowflake.ID]string {
	mentions := make(map[snowflake.ID]string, len(users))
	for _, user := range users {
		// we should fetch meber information to get the effective name
		// but to avoid unnecessary API calls, we can use the member cache.
		member, ok := client.Caches().Member(guildID, user.ID)
		if !ok {
			slog.Warn("Member not found in cache for mention", "mentionID", user.ID)
			mentions[user.ID] = user.EffectiveName()
		} else {
			mentions[user.ID] = member.EffectiveName()
		}
	}
	return mentions
}

func (s *Session) onJoinVoiceChannel(event *events.GuildVoiceStateUpdate) {
	voiceState := event.VoiceState
	// notify someone joined the voice channel
	slog.Info("User joined voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	// TODO: remove hardcoded message
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		preset, err := s.presetResolver.ResolveGuildPreset(ctx, event.Member.GuildID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(preset.Language))
		if !ok {
			slog.Warn("Voice resources not found for locale", "locale", preset.Language)
			return
		}

		s.worker.EnqueueTask(audio.NewSpeechTask(preset, []string{
			fmt.Sprintf(vr.Session.UserJoin, event.Member.EffectiveName()),
		}))
	}()
}

func (s *Session) onLeaveVoiceChannel(event *events.GuildVoiceStateUpdate) LeaveResult {
	voiceState := event.OldVoiceState

	// notify someone left the voice channel
	slog.Info("User left voice channel", "userID", voiceState.UserID, "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)

	if isVoiceChannelEmpty(event.Client().Rest(), event.Client().Caches(), voiceState.GuildID, *voiceState.ChannelID, voiceState.UserID) {
		slog.Info("Voice channel is empty, closing session", "guildID", voiceState.GuildID, "channelID", *voiceState.ChannelID)
		return LeaveResultClose
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		preset, err := s.presetResolver.ResolveGuildPreset(ctx, event.Member.GuildID)
		if err != nil {
			slog.Error("Failed to resolve preset", slog.Any("err", err))
			return
		}

		vr, ok := s.voiceResources.GetOrGeneric(discord.Locale(preset.Language))
		if !ok {
			slog.Warn("Voice resources not found for locale", "locale", preset.Language)
			return
		}

		s.worker.EnqueueTask(audio.NewSpeechTask(preset, []string{
			fmt.Sprintf(vr.Session.UserLeave, event.Member.EffectiveName()),
		}))
	}()

	return LeaveResultKeepAlive
}

func isVoiceChannelEmpty(
	client rest.Users,
	cache interface {
		cache.VoiceStateCache
		cache.MemberCache
	}, guildID, channelID, ignoredUserID snowflake.ID,
) bool {
	empty := true
	cache.VoiceStatesForEach(guildID, func(voiceState discord.VoiceState) {
		// ignore voice states of the user who left the voice channel
		if voiceState.UserID == ignoredUserID {
			return
		}

		// ignore bot

		user, ok := fetchUser(guildID, voiceState.UserID, client, cache)
		if !ok {
			slog.Warn("Failed to fetch user for voice state", "userID", voiceState.UserID, "guildID", guildID)
			return
		}

		if user.Bot {
			slog.Debug("Ignoring bot in voice channel", "userID", voiceState.UserID, "guildID", guildID)
			return
		}

		if voiceState.ChannelID != nil && *voiceState.ChannelID == channelID {
			empty = false
			return
		}
	})

	return empty
}

func fetchUser(guildID, userID snowflake.ID, client rest.Users, cache cache.MemberCache) (*discord.User, bool) {
	member, ok := cache.Member(guildID, userID)
	if ok {
		return &member.User, true
	}

	user, err := client.GetUser(userID)
	if err == nil {
		return user, true
	}

	return nil, false
}

func (s *Session) String() string {
	return fmt.Sprintf("Session(textChannelID: %s)", s.textChannelID)
}
