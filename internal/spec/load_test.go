package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taito.spec")
	content := `{"type": "skill", "name": "test-skill", "version": "1.0.0"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if s.Type != TypeSkill {
		t.Errorf("Type = %q, want %q", s.Type, TypeSkill)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", s.Version, "1.0.0")
	}
}

func TestLoadFromDirValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taito.spec")
	content := `{"type": "agent", "name": "test-agent"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	s, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	if s.Type != TypeAgent {
		t.Errorf("Type = %q, want %q", s.Type, TypeAgent)
	}
	if s.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", s.Name, "test-agent")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/taito.spec")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
}

func TestLoadFromDirNoSpecFile(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("Expected error when taito.spec is missing from directory, got nil")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taito.spec")
	if err := os.WriteFile(path, []byte(`{not valid json}`), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taito.spec")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Expected error for empty file, got nil")
	}
}

func TestLoadBundleWithIncludes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taito.spec")
	content := `{
		"type": "bundle",
		"name": "my-bundle",
		"includes": [
			"./skills/a/taito.spec",
			"./skills/b/taito.spec"
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if s.Type != TypeBundle {
		t.Errorf("Type = %q, want %q", s.Type, TypeBundle)
	}
	if len(s.Includes) != 2 {
		t.Fatalf("Includes length = %d, want 2", len(s.Includes))
	}
	if s.Includes[0] != "./skills/a/taito.spec" {
		t.Errorf("Includes[0] = %q", s.Includes[0])
	}
	if s.Includes[1] != "./skills/b/taito.spec" {
		t.Errorf("Includes[1] = %q", s.Includes[1])
	}
}

func TestLoadExampleFiles(t *testing.T) {
	// Load the actual example taito.spec files from the repository to ensure
	// they parse correctly with our data structures.
	examples := []struct {
		path         string
		expectedType string
		expectedName string
	}{
		{
			path:         "../../taito.spec/examples/bundle-repo/taito.spec",
			expectedType: TypeBundle,
			expectedName: "devtools-bundle",
		},
		{
			path:         "../../taito.spec/examples/bundle-repo/skills/git-commit-helper/taito.spec",
			expectedType: TypeSkill,
			expectedName: "git-commit-helper",
		},
		{
			path:         "../../taito.spec/examples/bundle-repo/skills/doc-generator/taito.spec",
			expectedType: TypeSkill,
			expectedName: "doc-generator",
		},
		{
			path:         "../../taito.spec/examples/bundle-repo/agents/devops-agent/taito.spec",
			expectedType: TypeAgent,
			expectedName: "devops-agent",
		},
	}

	for _, tc := range examples {
		t.Run(tc.expectedName, func(t *testing.T) {
			s, err := Load(tc.path)
			if err != nil {
				t.Fatalf("Load(%q) failed: %v", tc.path, err)
			}
			if s.Type != tc.expectedType {
				t.Errorf("Type = %q, want %q", s.Type, tc.expectedType)
			}
			if s.Name != tc.expectedName {
				t.Errorf("Name = %q, want %q", s.Name, tc.expectedName)
			}
		})
	}
}
