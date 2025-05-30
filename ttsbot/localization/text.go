package localization

import (
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"strings"
)

var textResources = make(map[string]TextResource)

type TextResource struct {
	Commands struct {
		Join struct {
			Description string `json:"description"` // format: "Start text-to-speech in text channels"`
		} `json:"join"`
		About struct {
			Description string `json:"description"` // format: "Display version of the bot"`
		} `json:"version"`
	} `json:"commands"`
	Options struct {
		Language struct {
			Description string `json:"description"` // format: "Language for text-to-speech."`
		} `json:"language"`
		Gender struct {
			Description string `json:"description"` // format: "gender of the voice for text-to-speech."`
			Male        string `json:"male"`        // format: "male"
			Female      string `json:"female"`      // format: "female"
			Neutral     string `json:"neutral"`     // format: "neutral"
		}
		Voice struct {
			Description string `json:"description"` // format: "Voice name for text-to-speech."`
		}
	}
}

func LoadTextResources(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip directories
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			// Skip non-JSON files
			continue
		}

		locale := strings.TrimSuffix(entry.Name(), ".json")

		filePath := path.Join(directory, entry.Name())

		file, err := os.Open(filePath)
		if err != nil {
			// skip file if it cannot be opened
			slog.Warn("failed to open text resource file", "file", filePath, "error", err)
			continue
		}
		defer file.Close()

		var resource TextResource
		err = json.NewDecoder(file).Decode(&resource)
		if err != nil {
			slog.Warn("failed to decode text resource file", "file", filePath, "error", err)
			continue
		}

		textResources[locale] = resource
		slog.Info("Loaded text resource", "locale", locale, "file", filePath)
	}

	return nil
}
