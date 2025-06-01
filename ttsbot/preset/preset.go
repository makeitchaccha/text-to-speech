package preset

import (
	"fmt"
)

type PresetID string

type Preset struct {
	Identifier   PresetID
	Engine       string
	Language     string
	VoiceName    string
	SpeakingRate float64
}

type PresetRegistry struct {
	presets map[PresetID]Preset // identifier -> Preset
	lists   []Preset
}

func NewPresetRegistry() *PresetRegistry {
	return &PresetRegistry{
		presets: make(map[PresetID]Preset),
	}
}

func (r *PresetRegistry) Register(preset Preset) error {
	if _, ok := r.presets[preset.Identifier]; ok {
		return fmt.Errorf("preset already registered: %s", preset.Identifier)
	}
	r.presets[preset.Identifier] = preset
	r.lists = append(r.lists, preset)

	return nil
}

func (r *PresetRegistry) Get(identifier PresetID) (Preset, bool) {
	preset, ok := r.presets[identifier]
	return preset, ok
}

func (r *PresetRegistry) List() []Preset {
	return r.lists
}
