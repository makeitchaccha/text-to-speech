package preset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/disgoorg/snowflake/v2"
)

type PresetResolver interface {
	Resolve(ctx context.Context, guildID, userID snowflake.ID) (Preset, error)
	ResolveGuildPreset(ctx context.Context, guildID snowflake.ID) (Preset, error)
}

func NewPresetResolver(registry *PresetRegistry, repository PresetIDRepository, fallbackPresetID PresetID) (PresetResolver, error) {
	// Validate the fallback preset ID exists in the registry
	if _, ok := registry.Get(fallbackPresetID); !ok {
		return nil, fmt.Errorf("fallback preset ID %s not found in registry", fallbackPresetID)
	}

	return &presetResolverImpl{
		registry:         registry,
		repository:       repository,
		fallbackPresetID: fallbackPresetID,
	}, nil
}

type presetResolverImpl struct {
	registry         *PresetRegistry
	repository       PresetIDRepository
	fallbackPresetID PresetID
}

func (r *presetResolverImpl) Resolve(ctx context.Context, guildID, userID snowflake.ID) (Preset, error) {
	presetID, err := r.resolveID(ctx, guildID, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// just log the error to notify about the issue, but use the fallback preset ID
			slog.Warn("failed to resolve preset ID", "guildID", guildID, "userID", userID, "error", err)
		}
		presetID = r.fallbackPresetID
	}
	preset, ok := r.registry.Get(presetID)
	if !ok {
		slog.Error("preset not found in registry", "presetID", presetID, "guildID", guildID, "userID", userID)
		return Preset{}, fmt.Errorf("preset not found for ID %s", presetID)
	}

	return preset, nil
}

func (r *presetResolverImpl) resolveID(ctx context.Context, guildID, userID snowflake.ID) (PresetID, error) {
	presetID, err := r.repository.Find(ctx, ScopeUser, userID)
	if err == nil {
		return presetID, nil
	}
	if errors.Is(err, ErrNotFound) {
		return "", err
	}

	presetID, err = r.repository.Find(ctx, ScopeGuild, guildID)
	if err == nil {
		return presetID, nil
	}
	if errors.Is(err, ErrNotFound) {
		return "", err
	}

	return "", ErrNotFound
}

func (r *presetResolverImpl) ResolveGuildPreset(ctx context.Context, guildID snowflake.ID) (Preset, error) {
	presetID, err := r.repository.Find(ctx, ScopeGuild, guildID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// just log the error to notify about the issue, but use the fallback preset ID
			slog.Warn("failed to resolve guild preset ID", "guildID", guildID, "error", err)
		}
		presetID = r.fallbackPresetID
	}

	preset, ok := r.registry.Get(presetID)
	if !ok {
		slog.Error("preset not found in registry for guild", "presetID", presetID, "guildID", guildID)
		return Preset{}, fmt.Errorf("preset not found for ID %s", presetID)
	}

	return preset, nil
}
