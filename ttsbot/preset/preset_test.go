package preset

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValidate(t *testing.T) {
	testcases := []struct {
		name    string
		preset  Preset
		wantErr bool
	}{
		{
			name: "valid preset",
			preset: Preset{
				Identifier: "test_preset",
				Engine:     "test_engine",
			},
			wantErr: false,
		},
		{
			name: "empty identifier",
			preset: Preset{
				Identifier: "",
				Engine:     "test_engine",
			},
			wantErr: true,
		},
		{
			name: "empty engine",
			preset: Preset{
				Identifier: "test_preset",
				Engine:     "",
			},
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.preset.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("preset.validate() = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	goodPreset := Preset{
		Identifier: "test_preset",
		Engine:     "test_engine",
	}

	badPreset := Preset{
		Identifier: "",
		Engine:     "test_engine",
	}

	testcases := []struct {
		name    string
		preset  Preset
		wantErr bool
	}{
		{
			name:    "register valid preset",
			preset:  goodPreset,
			wantErr: false,
		},
		{
			name:    "register invalid preset",
			preset:  badPreset,
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewPresetRegistry()
			err := registry.Register(tc.preset)
			if (err != nil) != tc.wantErr {
				t.Errorf("PresetRegistry.Register() = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGet(t *testing.T) {
	registry := NewPresetRegistry()
	preset := Preset{
		Identifier: "test_preset",
		Engine:     "test_engine",
	}

	if err := registry.Register(preset); err != nil {
		t.Fatalf("Failed to register preset: %v", err)
	}

	retrieved, ok := registry.Get(preset.Identifier)
	if !ok {
		t.Fatalf("registry.Get() = _, false, want true for identifier %s", preset.Identifier)
	}

	if !cmp.Equal(retrieved, preset) {
		t.Errorf("registry.Get() = %v, _, want %v", retrieved, preset)
	}
}
