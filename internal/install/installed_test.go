package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInstalledMissingFile(t *testing.T) {
	origDir := os.Getenv("XDG_CONFIG_HOME")
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() {
		if origDir != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origDir)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if len(idx.Installed.Skills) != 0 || len(idx.Installed.Agents) != 0 || len(idx.Installed.Bundles) != 0 {
		t.Errorf("expected 0 entries")
	}
}

func TestSaveAndLoadInstalled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	idx := &InstalledIndex{
		Version: IndexSpecVersion,
		Installed: InstalledPackages{
			Skills: []InstalledEntry{
				{
					Name:        "git-helper",
					SpecType:    "skill",
					Reference:   "ghcr.io/org/git-helper:1.0.0",
					InstallIn:   []InstallLocation{{Tool: "cursor", Path: "/home/user/.cursor/skills/git-helper"}},
					InstalledAt: "2026-04-06T12:00:00Z",
				},
			},
		},
	}

	if err := SaveInstalled(idx); err != nil {
		t.Fatalf("SaveInstalled: %v", err)
	}

	loaded, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if len(loaded.Installed.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Installed.Skills))
	}
	e := loaded.Installed.Skills[0]
	if e.Name != "git-helper" {
		t.Errorf("Name = %q, want %q", e.Name, "git-helper")
	}
}

func TestUpsertEntry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	err := UpsertEntry(InstalledEntry{
		Name:      "git-helper",
		SpecType:  "skill",
		Reference: "ghcr.io/org/git-helper:1.0.0",
		InstallIn: []InstallLocation{{Tool: "cursor", Path: "/home/user/.cursor/skills/git-helper"}},
	})
	if err != nil {
		t.Fatalf("UpsertEntry 1: %v", err)
	}

	err = UpsertEntry(InstalledEntry{
		Name:      "git-helper",
		SpecType:  "skill",
		Reference: "ghcr.io/org/git-helper:1.0.0",
		InstallIn: []InstallLocation{{Tool: "claude-code", Path: "/home/user/.claude/skills/git-helper"}},
	})
	if err != nil {
		t.Fatalf("UpsertEntry 2: %v", err)
	}

	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if len(idx.Installed.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Installed.Skills))
	}
	if len(idx.Installed.Skills[0].InstallIn) != 2 {
		t.Fatalf("expected 2 install locations, got %d", len(idx.Installed.Skills[0].InstallIn))
	}

	err = UpsertEntry(InstalledEntry{
		Name:      "git-helper",
		SpecType:  "skill",
		Reference: "ghcr.io/org/git-helper:2.0.0",
		InstallIn: []InstallLocation{{Tool: "cursor", Path: "/home/user/.cursor/skills/git-helper"}},
	})
	if err != nil {
		t.Fatalf("UpsertEntry 3: %v", err)
	}

	idx, err = LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if len(idx.Installed.Skills) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(idx.Installed.Skills))
	}
	if len(idx.Installed.Skills[0].InstallIn) != 2 {
		t.Fatalf("expected 2 install locations after upsert, got %d", len(idx.Installed.Skills[0].InstallIn))
	}
}

func TestUpsertBundle(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	_, err := UpsertBundle(BundleEntry{
		Name:     "my-bundle",
		SpecType: "bundle",
		Version:  "1.0",
	})
	if err != nil {
		t.Fatalf("UpsertBundle 1: %v", err)
	}

	idx, _ := LoadInstalled()
	if len(idx.Installed.Bundles) != 1 {
		t.Fatalf("expected 1 bundle, got %d", len(idx.Installed.Bundles))
	}

	_, err = UpsertBundle(BundleEntry{
		Name:     "my-bundle",
		SpecType: "bundle",
		Version:  "2.0",
	})
	if err != nil {
		t.Fatalf("UpsertBundle update: %v", err)
	}
	idx, _ = LoadInstalled()
	if len(idx.Installed.Bundles) != 1 || idx.Installed.Bundles[0].Version != "2.0" {
		t.Fatalf("UpsertBundle update failed")
	}
}

func TestUninstallSingleSkill(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir1 := filepath.Join(tmp, "tool1", "skills", "my-skill")
	_ = os.MkdirAll(dir1, 0755)

	_ = UpsertEntry(InstalledEntry{Name: "my-skill", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor", Path: dir1}}})
	_ = UpsertEntry(InstalledEntry{Name: "other", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor", Path: "/some/path"}}})

	idx, _ := LoadInstalled()
	if len(idx.Installed.Skills) != 1 {
		t.Fatalf("expected 1 skill")
	}
	id := idx.Installed.Skills[0].ID

	results, err := Uninstall(id)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	idx, _ = LoadInstalled()
	if len(idx.Installed.Skills) != 0 {
		t.Fatalf("expected 0 remaining skills, got %d", len(idx.Installed.Skills))
	}
	if len(idx.Installed.Agents) != 1 {
		t.Fatalf("expected 1 remaining agent, got %d", len(idx.Installed.Agents))
	}
}

func TestUninstallBundleRemovesChildren(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	skillDir := filepath.Join(tmp, "tool", "skills", "child-skill")
	_ = os.MkdirAll(skillDir, 0755)

	bundleID, _ := UpsertBundle(BundleEntry{Name: "my-bundle", SpecType: "bundle"})
	_ = UpsertEntry(InstalledEntry{
		Name:      "child-skill",
		SpecType:  "skill",
		BundleID:  bundleID,
		InstallIn: []InstallLocation{{Tool: "cursor", Path: skillDir}},
	})

	results, err := Uninstall(bundleID)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	idx, _ := LoadInstalled()
	if len(idx.Installed.Skills) != 0 {
		t.Fatalf("expected 0 remaining skills")
	}
	if len(idx.Installed.Bundles) != 0 {
		t.Fatalf("expected 0 remaining bundles")
	}
}

func TestBundleIDOmittedWhenEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	_ = UpsertEntry(InstalledEntry{
		Name:      "standalone-skill",
		SpecType:  "skill",
		InstallIn: []InstallLocation{{Tool: "cursor"}},
	})

	p := filepath.Join(tmp, "taito", "installed.json")
	data, _ := os.ReadFile(p)
	if strings.Contains(string(data), "bundleId") {
		t.Error("bundleId should be omitted from JSON when empty")
	}
}

func TestUUIDGeneration(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Test UpsertEntry generates an ID
	err := UpsertEntry(InstalledEntry{
		Name:      "test-skill",
		SpecType:  "skill",
		InstallIn: []InstallLocation{{Tool: "cursor"}},
	})
	if err != nil {
		t.Fatalf("UpsertEntry failed: %v", err)
	}

	idx, _ := LoadInstalled()
	if len(idx.Installed.Skills) != 1 {
		t.Fatalf("Expected 1 skill")
	}
	id1 := idx.Installed.Skills[0].ID
	if id1 == "" {
		t.Fatalf("Expected non-empty ID for new skill")
	}

	// Test updating preserves ID
	err = UpsertEntry(InstalledEntry{
		Name:      "test-skill",
		SpecType:  "skill",
		InstallIn: []InstallLocation{{Tool: "cursor"}},
		Version:   "2.0",
	})
	if err != nil {
		t.Fatalf("UpsertEntry update failed: %v", err)
	}

	idx, _ = LoadInstalled()
	if len(idx.Installed.Skills) != 1 {
		t.Fatalf("Expected 1 skill after update")
	}
	if idx.Installed.Skills[0].ID != id1 {
		t.Fatalf("Expected ID to be preserved (%s), got %s", id1, idx.Installed.Skills[0].ID)
	}
	if idx.Installed.Skills[0].Version != "2.0" {
		t.Fatalf("Expected version to be updated")
	}

	// Test Bundle generating ID
	bundleID, err := UpsertBundle(BundleEntry{
		Name:     "test-bundle",
		SpecType: "bundle",
	})
	if err != nil {
		t.Fatalf("UpsertBundle failed: %v", err)
	}

	idx, _ = LoadInstalled()
	if len(idx.Installed.Bundles) != 1 {
		t.Fatalf("Expected 1 bundle")
	}
	if bundleID == "" || bundleID != idx.Installed.Bundles[0].ID {
		t.Fatalf("Expected valid ID for new bundle")
	}
	if bundleID == id1 {
		t.Fatalf("Bundle ID should be unique")
	}
}
