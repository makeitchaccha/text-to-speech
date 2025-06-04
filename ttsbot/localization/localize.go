package localization

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"
)

func load[S ~string, T any, U ~map[S]T](directory string, resources U) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("failed to read text resources directory: %w", err)
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
			return fmt.Errorf("failed to open text resource file %s: %w", filePath, err)
		}
		defer file.Close()

		var resource T
		metadata, err := toml.NewDecoder(file).Decode(&resource)
		if err != nil {
			return fmt.Errorf("failed to decode text resource file %s: %w", filePath, err)
		}

		if len(metadata.Undecoded()) > 0 {
			slog.Warn("text resource file contains undecoded fields", "file", filePath, "fields", metadata.Undecoded())
			return fmt.Errorf("text resource file %s contains undecoded fields: %v", filePath, metadata.Undecoded())
		}

		resources[S(locale)] = resource
		slog.Info("Loaded text resource", "locale", locale, "file", filePath)
	}

	return nil
}
