package spec

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalFullSpec(t *testing.T) {
	input := `{
		"taitoVersion": "0.1.0",
		"type": "skill",
		"name": "git-commit-helper",
		"version": "1.2.0",
		"description": "Helps write conventional commit messages",
		"source": "github.com/taito-project/git-commit-helper",
		"author": {
			"name": "Taito",
			"url": "https://github.com/taito-project",
			"email": "team@taito.dev"
		},
		"license": "Apache-2.0",
		"keywords": ["git", "commit", "developer-tools"]
	}`

	var s TaitoSpec
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if s.TaitoVersion != "0.1.0" {
		t.Errorf("TaitoVersion = %q, want %q", s.TaitoVersion, "0.1.0")
	}
	if s.Type != TypeSkill {
		t.Errorf("Type = %q, want %q", s.Type, TypeSkill)
	}
	if s.Name != "git-commit-helper" {
		t.Errorf("Name = %q, want %q", s.Name, "git-commit-helper")
	}
	if s.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", s.Version, "1.2.0")
	}
	if s.Description != "Helps write conventional commit messages" {
		t.Errorf("Description = %q", s.Description)
	}
	if s.Source != "github.com/taito-project/git-commit-helper" {
		t.Errorf("Source = %q", s.Source)
	}
	if s.Author == nil {
		t.Fatal("Author is nil")
	}
	if s.Author.Name != "Taito" {
		t.Errorf("Author.Name = %q", s.Author.Name)
	}
	if s.Author.URL != "https://github.com/taito-project" {
		t.Errorf("Author.URL = %q", s.Author.URL)
	}
	if s.Author.Email != "team@taito.dev" {
		t.Errorf("Author.Email = %q", s.Author.Email)
	}
	if s.License != "Apache-2.0" {
		t.Errorf("License = %q", s.License)
	}
	if len(s.Keywords) != 3 {
		t.Fatalf("Keywords length = %d, want 3", len(s.Keywords))
	}
	if s.Keywords[0] != "git" || s.Keywords[1] != "commit" || s.Keywords[2] != "developer-tools" {
		t.Errorf("Keywords = %v", s.Keywords)
	}
}

func TestUnmarshalMinimalSpec(t *testing.T) {
	input := `{"type": "skill", "name": "my-skill"}`

	var s TaitoSpec
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if s.Type != TypeSkill {
		t.Errorf("Type = %q, want %q", s.Type, TypeSkill)
	}
	if s.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "my-skill")
	}
	// All optional fields should be zero values.
	if s.TaitoVersion != "" {
		t.Errorf("TaitoVersion = %q, want empty", s.TaitoVersion)
	}
	if s.Author != nil {
		t.Errorf("Author = %v, want nil", s.Author)
	}
	if s.Keywords != nil {
		t.Errorf("Keywords = %v, want nil", s.Keywords)
	}
	if s.Includes != nil {
		t.Errorf("Includes = %v, want nil", s.Includes)
	}
}

func TestUnmarshalBundle(t *testing.T) {
	input := `{
		"type": "bundle",
		"name": "devtools",
		"includes": [
			"./skills/git-helper/taito.spec",
			"./agents/devops-agent/taito.spec"
		]
	}`

	var s TaitoSpec
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if s.Type != TypeBundle {
		t.Errorf("Type = %q, want %q", s.Type, TypeBundle)
	}
	if len(s.Includes) != 2 {
		t.Fatalf("Includes length = %d, want 2", len(s.Includes))
	}
	if s.Includes[0] != "./skills/git-helper/taito.spec" {
		t.Errorf("Includes[0] = %q", s.Includes[0])
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	original := TaitoSpec{
		TaitoVersion: "0.1.0",
		Type:         TypeAgent,
		Name:         "devops-agent",
		Version:      "1.0.0",
		Description:  "A DevOps agent",
		Author: &Author{
			Name: "Taito",
			URL:  "https://github.com/taito-project",
		},
		License:  "MIT",
		Keywords: []string{"devops", "agent"},
	}

	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored TaitoSpec
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.Type != original.Type {
		t.Errorf("Type = %q, want %q", restored.Type, original.Type)
	}
	if restored.Name != original.Name {
		t.Errorf("Name = %q, want %q", restored.Name, original.Name)
	}
	if restored.Version != original.Version {
		t.Errorf("Version = %q, want %q", restored.Version, original.Version)
	}
	if restored.Author == nil || restored.Author.Name != original.Author.Name {
		t.Errorf("Author round-trip failed")
	}
	if len(restored.Keywords) != len(original.Keywords) {
		t.Errorf("Keywords length = %d, want %d", len(restored.Keywords), len(original.Keywords))
	}
}

func TestUnmarshalIgnoresUnknownFields(t *testing.T) {
	// Verify that unknown JSON fields (like x- extensions or removed fields)
	// are silently ignored.
	input := `{
		"type": "skill",
		"name": "my-skill",
		"x-custom-field": "custom-value",
		"config": {"API_KEY": {"type": "string"}},
		"repository": {"type": "git", "url": "https://example.com"}
	}`

	var s TaitoSpec
	if err := json.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal should not fail on unknown fields: %v", err)
	}

	if s.Type != TypeSkill {
		t.Errorf("Type = %q, want %q", s.Type, TypeSkill)
	}
	if s.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "my-skill")
	}
}

func TestMarshalOmitsEmptyFields(t *testing.T) {
	s := TaitoSpec{
		Type: TypeSkill,
		Name: "minimal",
	}

	data, err := json.Marshal(&s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Re-parse as a generic map to inspect which keys are present.
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	// Only "type" and "name" should be present.
	expected := map[string]bool{"type": true, "name": true}
	for k := range m {
		if !expected[k] {
			t.Errorf("Unexpected key %q in marshalled output", k)
		}
	}
	for k := range expected {
		if _, ok := m[k]; !ok {
			t.Errorf("Expected key %q missing from marshalled output", k)
		}
	}
}
