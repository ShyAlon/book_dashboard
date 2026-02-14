package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const BaseDirName = "ManuscriptHealth"

type Settings struct {
	Theme        string `json:"theme"`
	DefaultModel string `json:"default_model"`
}

func EnsureDefault() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	return EnsureAt(filepath.Join(home, BaseDirName))
}

func EnsureAt(base string) (string, error) {
	paths := []string{
		filepath.Join(base, "configs"),
		filepath.Join(base, "cache", "embeddings"),
		filepath.Join(base, "projects"),
	}

	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", p, err)
		}
	}

	settingsPath := filepath.Join(base, "configs", "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		defaults := Settings{
			Theme:        "darkroom",
			DefaultModel: "llama3",
		}
		raw, marshalErr := json.MarshalIndent(defaults, "", "  ")
		if marshalErr != nil {
			return "", fmt.Errorf("marshal settings: %w", marshalErr)
		}
		if writeErr := os.WriteFile(settingsPath, raw, 0o644); writeErr != nil {
			return "", fmt.Errorf("write settings: %w", writeErr)
		}
	}

	return base, nil
}
