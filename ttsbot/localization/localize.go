package localization

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
)

type genericResources[S ~string, T any] map[S]T

func (r genericResources[S, T]) Get(locale S) (T, bool) {
	resource, ok := r[locale]
	return resource, ok
}

// This method returns the resource following order.
//  1. If the locale exists, return it.
//  2. If the locale does not exist, return the generic resource for the locale if it exists.
//     For example, given a locale "en-US" but there is no resource for "en-US",
//     then try to return the resource for "en" if it exists.
//  3. If the generic resource does not exist, return no resource.
//
// TODO: profile this method to see if it is a bottleneck.
// If it is, consider caching the generic resources in a map.
func (r genericResources[S, T]) GetOrGeneric(locale S) (T, bool) {
	resource, ok := r.Get(locale)
	if ok {
		return resource, true
	}
	genericLocale := S(strings.SplitN(string(locale), "-", 2)[0])
	genericResource, ok := r.Get(genericLocale)
	if ok {
		return genericResource, true
	}

	var zero T
	return zero, false
}

func (r genericResources[S, T]) Localizations(value func(resource T) string) map[S]string {
	localizations := make(map[S]string, len(r))
	for locale, resource := range r {
		localizations[locale] = value(resource)
	}
	return localizations
}

func load[S ~string, T any, U genericResources[S, T]](directory string, resources U) error {
	var resource T
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to read %T resources directory: %w", resource, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip directories
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".toml") {
			// Skip non-TOML files
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".toml")

		filePath := path.Join(directory, entry.Name())

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open %T resource file %s: %w", resource, filePath, err)
		}
		defer file.Close()

		metadata, err := toml.NewDecoder(file).Decode(&resource)
		if err != nil {
			return fmt.Errorf("failed to decode %T resource file %s: %w", resource, filePath, err)
		}

		if len(metadata.Undecoded()) > 0 {
			slog.Warn("The resource file contains undecoded fields", "file", filePath, "fields", metadata.Undecoded())
			return fmt.Errorf("%T resource file %s contains undecoded fields: %v", resource, filePath, metadata.Undecoded())
		}

		resources[S(locale)] = resource
		slog.Info("Loaded the resource", "locale", locale, "file", filePath)
	}

	return nil
}
