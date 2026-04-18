package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const manifestFilename = "taito.spec"

// Load reads a taito.spec file at the given path and returns a parsed TaitoSpec.
// It returns an error if the file cannot be read or contains invalid JSON.
func Load(path string) (*TaitoSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading taito.spec: %w", err)
	}

	var s TaitoSpec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing taito.spec: %w", err)
	}

	return &s, nil
}

// LoadFromDir finds and loads the taito.spec file in the given directory.
// It looks for a file named "taito.spec" at the root of dir.
func LoadFromDir(dir string) (*TaitoSpec, error) {
	path := filepath.Join(dir, manifestFilename)
	return Load(path)
}
