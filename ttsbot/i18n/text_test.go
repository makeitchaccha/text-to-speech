package i18n

import (
	"fmt"
	"testing"
)

func TestLoadTextResources(t *testing.T) {
	trs, err := LoadTextResources("../../locales/text/", "en-US")
	if err != nil {
		t.Fatalf("Failed to load text resources: %v", err)
	}

	if len(trs.genericResources) == 0 {
		t.Fatal("No text resources loaded")
	}

	for locale, resource := range trs.genericResources {
		t.Run(fmt.Sprintf("locale_%s", locale), func(t *testing.T) {
			errs := validateResource(resource, "TextResource")
			if len(errs) > 0 {
				for _, e := range errs {
					t.Error(e)
				}
			}
		})
	}
}
