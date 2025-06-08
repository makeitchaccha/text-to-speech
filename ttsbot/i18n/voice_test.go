package i18n

import (
	"fmt"
	"testing"
)

func TestLoadVoiceResources(t *testing.T) {
	trs, err := LoadVoiceResources("../../locales/voice/")
	if err != nil {
		t.Fatalf("Failed to load voice resources: %v", err)
	}

	if len(trs.genericResources) == 0 {
		t.Fatal("No voice resources loaded")
	}

	for locale, resource := range trs.genericResources {
		t.Run(fmt.Sprintf("locale_%s", locale), func(t *testing.T) {
			errs := validateResource(resource, "VoiceResource")
			if len(errs) > 0 {
				for _, e := range errs {
					t.Error(e)
				}
			}
		})
	}
}
