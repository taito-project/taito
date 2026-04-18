package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHashReference(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{"full reference with tag", "registry.gitlab.com/skill-harbor/infrastructure/test:v1.0.0"},
		{"full reference no tag", "registry.gitlab.com/skill-harbor/infrastructure/test"},
		{"ghcr reference", "ghcr.io/org/my-skill:1.0.0"},
		{"localhost with port", "localhost:5000/myskill:latest"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := HashReference(tc.ref)
			if len(h) != 32 {
				t.Errorf("HashReference(%q) length = %d, want 32", tc.ref, len(h))
			}
			// Should be deterministic.
			h2 := HashReference(tc.ref)
			if h != h2 {
				t.Errorf("HashReference not deterministic: %q != %q", h, h2)
			}
		})
	}
}

func TestHashReferenceDifferentInputsDifferentHashes(t *testing.T) {
	a := HashReference("ghcr.io/org-a/test:1.0.0")
	b := HashReference("ghcr.io/org-b/test:1.0.0")
	if a == b {
		t.Errorf("different references should produce different hashes: %q", a)
	}
}

func TestHashReferenceNoNormalization(t *testing.T) {
	// "foo/bar" and "foo/bar:latest" are NOT normalized — they produce different hashes.
	a := HashReference("registry.example.com/org/test")
	b := HashReference("registry.example.com/org/test:latest")
	if a == b {
		t.Error("tagless and :latest references should produce different hashes (no normalization)")
	}
}

func TestPackagePath(t *testing.T) {
	cacheDir := "/home/user/.cache/taito"
	ref := "ghcr.io/org/my-skill:1.0.0"

	path := PackagePath(cacheDir, ref)
	hash := HashReference(ref)
	expected := filepath.Join(cacheDir, "packages", hash)

	if path != expected {
		t.Errorf("PackagePath = %q, want %q", path, expected)
	}
}

func TestLoadIndexMissingFile(t *testing.T) {
	cacheDir := t.TempDir()

	idx, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(idx.Entries))
	}
}

func TestSaveAndLoadIndex(t *testing.T) {
	cacheDir := t.TempDir()

	idx := &Index{
		Entries: map[string]IndexEntry{
			"abc123": {
				Reference: "ghcr.io/org/test:1.0.0",
				SpecType:  "skill",
				Format:    "oci",
				CreatedAt: "2026-04-04T12:00:00Z",
			},
		},
	}

	if err := SaveIndex(cacheDir, idx); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// Verify the file exists.
	p := filepath.Join(cacheDir, "packages", "index.json")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("index.json not created: %v", err)
	}

	// Load it back.
	loaded, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	entry := loaded.Entries["abc123"]
	if entry.Reference != "ghcr.io/org/test:1.0.0" {
		t.Errorf("Reference = %q, want %q", entry.Reference, "ghcr.io/org/test:1.0.0")
	}
	if entry.SpecType != "skill" {
		t.Errorf("SpecType = %q, want %q", entry.SpecType, "skill")
	}
}

func TestAddEntry(t *testing.T) {
	cacheDir := t.TempDir()

	ref := "ghcr.io/org/my-skill:1.0.0"
	if err := AddEntry(cacheDir, ref, "agent", "oci"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	idx, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	hash := HashReference(ref)
	entry, ok := idx.Entries[hash]
	if !ok {
		t.Fatalf("entry not found for hash %q", hash)
	}
	if entry.Reference != ref {
		t.Errorf("Reference = %q, want %q", entry.Reference, ref)
	}
	if entry.SpecType != "agent" {
		t.Errorf("SpecType = %q, want %q", entry.SpecType, "agent")
	}
	if entry.Format != "oci" {
		t.Errorf("Format = %q, want %q", entry.Format, "oci")
	}
	if entry.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestAddEntryUpserts(t *testing.T) {
	cacheDir := t.TempDir()

	ref := "ghcr.io/org/my-skill:1.0.0"
	if err := AddEntry(cacheDir, ref, "skill", "oci"); err != nil {
		t.Fatalf("AddEntry 1: %v", err)
	}
	// Upsert with new specType.
	if err := AddEntry(cacheDir, ref, "agent", "oci"); err != nil {
		t.Fatalf("AddEntry 2: %v", err)
	}

	idx, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}

	if len(idx.Entries) != 1 {
		t.Errorf("expected 1 entry after upsert, got %d", len(idx.Entries))
	}
	hash := HashReference(ref)
	if idx.Entries[hash].SpecType != "agent" {
		t.Errorf("SpecType should be updated to 'agent', got %q", idx.Entries[hash].SpecType)
	}
}

func TestRemoveEntry(t *testing.T) {
	cacheDir := t.TempDir()

	ref := "ghcr.io/org/my-skill:1.0.0"
	if err := AddEntry(cacheDir, ref, "skill", "oci"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	hash := HashReference(ref)
	if err := RemoveEntry(cacheDir, hash); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}

	idx, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected 0 entries after remove, got %d", len(idx.Entries))
	}
}

func TestClearIndex(t *testing.T) {
	cacheDir := t.TempDir()

	if err := AddEntry(cacheDir, "ref1", "skill", "oci"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if err := AddEntry(cacheDir, "ref2", "agent", "oci"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	if err := ClearIndex(cacheDir); err != nil {
		t.Fatalf("ClearIndex: %v", err)
	}

	// File should be gone.
	p := filepath.Join(cacheDir, "packages", "index.json")
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("index.json should be removed after ClearIndex")
	}

	// LoadIndex should still work (returns empty).
	idx, err := LoadIndex(cacheDir)
	if err != nil {
		t.Fatalf("LoadIndex after clear: %v", err)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(idx.Entries))
	}
}

func TestClearIndexNoFile(t *testing.T) {
	cacheDir := t.TempDir()

	// Should not error if the file doesn't exist.
	if err := ClearIndex(cacheDir); err != nil {
		t.Fatalf("ClearIndex on missing file: %v", err)
	}
}

func TestIndexJsonFormat(t *testing.T) {
	cacheDir := t.TempDir()

	if err := AddEntry(cacheDir, "ghcr.io/org/test:1.0.0", "skill", "oci"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	p := filepath.Join(cacheDir, "packages", "index.json")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Verify it's valid JSON with expected structure.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := raw["entries"]; !ok {
		t.Error("index.json should have 'entries' key")
	}
}
