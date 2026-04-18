package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Save writes the given Config to the standard config directory as
// config.json. It creates the directory if it does not exist.
func Save(cfg *Config) error {
	configDir, err := Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Append a trailing newline for cleaner diffs / editor display.
	data = append(data, '\n')

	return os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644)
}

// ConfigFilePath returns the full path to the config.json file.
func ConfigFilePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
