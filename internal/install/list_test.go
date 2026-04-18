package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestListSortOrder(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	idx := &InstalledIndex{
		Version: IndexSpecVersion,
		Installed: InstalledPackages{
			Skills: []InstalledEntry{
				{Name: "alpha", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "windsurf"}}, Version: "2.0.0", Reference: "ref/alpha"},
				{Name: "beta", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "0.1.0", Reference: "ref/beta"},
			},
			Agents: []InstalledEntry{
				{Name: "zebra", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0", Reference: "ref/zebra"},
				{Name: "alpha", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}, {Tool: "windsurf"}}, Version: "2.0.0", Reference: "ref/alpha"},
			},
		},
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(tmp, "taito")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "installed.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}

	var entries []InstalledEntry
	entries = append(entries, loaded.Installed.Skills...)
	entries = append(entries, loaded.Installed.Agents...)
	SortEntries(entries)

	expected := []struct {
		name     string
		specType string
		tools    int
	}{
		{"alpha", "agent", 2},
		{"alpha", "skill", 1},
		{"beta", "skill", 1},
		{"zebra", "agent", 1},
	}

	if len(entries) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(entries))
	}
	for i, e := range entries {
		if e.Name != expected[i].name || e.SpecType != expected[i].specType || len(e.InstallIn) != expected[i].tools {
			t.Errorf("entry[%d] = {%s, %s, len=%d}, want {%s, %s, len=%d}",
				i, e.Name, e.SpecType, len(e.InstallIn),
				expected[i].name, expected[i].specType, expected[i].tools)
		}
	}
}

func TestBuildTreeRowsStandaloneOnly(t *testing.T) {
	installed := InstalledPackages{
		Skills: []InstalledEntry{
			{Name: "beta", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0", Reference: "ref/beta"},
		},
		Agents: []InstalledEntry{
			{Name: "alpha", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "2.0.0", Reference: "ref/alpha"},
		},
	}

	rows := BuildTreeRows(installed, map[string]string{})

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][1] != "ref/alpha" {
		t.Errorf("row[0] name = %q, want %q", rows[0][1], "ref/alpha")
	}
	if rows[1][1] != "ref/beta" {
		t.Errorf("row[1] name = %q, want %q", rows[1][1], "ref/beta")
	}
}

func TestBuildTreeRowsBundleGrouping(t *testing.T) {
	installed := InstalledPackages{
		Skills: []InstalledEntry{
			{Name: "child-b", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0", BundleID: "b1"},
		},
		Agents: []InstalledEntry{
			{Name: "standalone", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0"},
			{Name: "child-a", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0", BundleID: "b1"},
		},
		Bundles: []BundleEntry{
			{ID: "b1", Name: "my-bundle", SpecType: "bundle", Version: "1.0.0"},
		},
	}

	rows := BuildTreeRows(installed, map[string]string{})

	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}

	if rows[0][1] != "" {
		t.Errorf("row[0] name = %q, want %q", rows[0][1], "")
	}

	if rows[1][1] != "" {
		t.Errorf("row[1] name = %q, want %q", rows[1][1], "")
	}
	if rows[1][2] != "bundle" {
		t.Errorf("row[1] type = %q, want %q", rows[1][2], "bundle")
	}

	if rows[2][1] != " ├─ child-a" {
		t.Errorf("row[2] name = %q, want %q", rows[2][1], " ├─ child-a")
	}

	if rows[3][1] != " ╰─ child-b" {
		t.Errorf("row[3] name = %q, want %q", rows[3][1], " ╰─ child-b")
	}
}

func TestBuildTreeRowsMultipleBundles(t *testing.T) {
	installed := InstalledPackages{
		Bundles: []BundleEntry{
			{ID: "b1", Name: "z-bundle", SpecType: "bundle"},
			{ID: "b2", Name: "a-bundle", SpecType: "bundle"},
		},
		Skills: []InstalledEntry{
			{Name: "c1", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, BundleID: "b1"},
			{Name: "c3", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, BundleID: "b2"},
		},
		Agents: []InstalledEntry{
			{Name: "c2", SpecType: "agent", InstallIn: []InstallLocation{{Tool: "cursor"}}, BundleID: "b2"},
		},
	}

	rows := BuildTreeRows(installed, map[string]string{})

	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}

	if rows[0][1] != "" {
		t.Errorf("row[0] = %q, want %q", rows[0][1], "")
	}
	if rows[3][1] != "" {
		t.Errorf("row[3] = %q, want %q", rows[3][1], "")
	}
}

func TestBuildTreeRowsToolDisplayNames(t *testing.T) {
	installed := InstalledPackages{
		Skills: []InstalledEntry{
			{Name: "my-skill", SpecType: "skill", InstallIn: []InstallLocation{{Tool: "cursor"}}, Version: "1.0.0"},
		},
	}

	displayNames := map[string]string{"cursor": "Cursor"}
	rows := BuildTreeRows(installed, displayNames)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][3] != "Cursor" {
		t.Errorf("tool = %q, want %q", rows[0][3], "Cursor")
	}
}
