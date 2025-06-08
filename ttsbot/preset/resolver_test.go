package preset

import (
	"context"
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func TestNewPresetResolver(t *testing.T) {
	testcases := []struct {
		name       string
		presets    []Preset
		fallbackID PresetID
		wantErr    bool
	}{
		{
			name: "valid presets with fallback",
			presets: []Preset{
				{Identifier: "sample_user_preset", Engine: "test_engine"},
				{Identifier: "sample_guild_preset", Engine: "test_engine"},
				{Identifier: "fallback_preset", Engine: "test_engine"},
			},
			fallbackID: "fallback_preset",
			wantErr:    false,
		},
		{
			name: "missing fallback preset",
			presets: []Preset{
				{Identifier: "sample_user_preset", Engine: "test_engine"},
				{Identifier: "sample_guild_preset", Engine: "test_engine"},
			},
			fallbackID: "fallback_preset",
			wantErr:    true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewPresetRegistry()
			for _, preset := range tc.presets {
				if err := registry.Register(preset); err != nil {
					t.Fatalf("failed to register preset: %v", err)
				}
			}

			repo := struct {
				PresetIDRepository
			}{}
			_, err := NewPresetResolver(registry, repo, tc.fallbackID)

			if (err != nil) != tc.wantErr {
				t.Errorf("NewPresetResolver() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
		})
	}
}

type FindStub struct {
	PresetIDRepository
}

func (f *FindStub) Find(_ context.Context, scope Scope, id snowflake.ID) (PresetID, error) {
	if scope == ScopeUser && id == 10 {
		return "sample_user_preset", nil
	} else if scope == ScopeGuild && id == 20 {
		return "sample_guild_preset", nil
	}
	return "", ErrNotFound
}

func TestResolve(t *testing.T) {
	registry := NewPresetRegistry()
	presets := []Preset{
		{Identifier: "sample_user_preset", Engine: "test_engine"},
		{Identifier: "sample_guild_preset", Engine: "test_engine"},
		{Identifier: "fallback_preset", Engine: "test_engine"},
	}
	for _, preset := range presets {
		if err := registry.Register(preset); err != nil {
			t.Fatalf("failed to register preset: %v", err)
		}
	}

	repo := &FindStub{}
	resolver, err := NewPresetResolver(registry, repo, "fallback_preset")
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	testcases := []struct {
		name    string
		guildID snowflake.ID
		userID  snowflake.ID
		wantID  PresetID
	}{
		{
			name:    "resolve user preset",
			guildID: 0,
			userID:  10, // user ID for which a preset exists
			wantID:  "sample_user_preset",
		},
		{
			name:    "resolve guild preset",
			guildID: 20, // guild ID for which a preset exists
			userID:  0,
			wantID:  "sample_guild_preset",
		},
		{
			name:    "resolve fallback preset",
			guildID: 0, // no preset for this guild
			userID:  0, // no preset for this user also
			wantID:  "fallback_preset",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			preset, err := resolver.Resolve(context.Background(), tc.guildID, tc.userID)
			if err != nil {
				t.Errorf("Resolve() error = %v, no error expected", err)
				return
			}
			if preset.Identifier != tc.wantID {
				t.Errorf("Resolve() got = %v, want %v", preset.Identifier, tc.wantID)
			}
		})
	}
}
