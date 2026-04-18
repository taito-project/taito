package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/taito-project/taito/internal/install"
)

func TestListCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'list' command to be registered on rootCmd")
	}
}

func TestListCmdHasAlias(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "list" {
			aliases := c.Aliases
			found := false
			for _, a := range aliases {
				if a == "ls" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected 'ls' alias, got aliases: %v", aliases)
			}
			return
		}
	}
	t.Error("list command not found")
}

func TestListVersionFieldPresent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	idx := &install.InstalledIndex{
		Version: install.IndexSpecVersion,
		Installed: install.InstalledPackages{
			Skills: []install.InstalledEntry{
				{
					Name:      "my-skill",
					SpecType:  "skill",
					Version:   "3.2.1",
					Reference: "ghcr.io/org/my-skill:3.2.1",
					InstallIn: []install.InstallLocation{{Tool: "cursor", Path: "/home/user/.cursor/skills/my-skill"}},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(idx, "", "  ")
	configDir := filepath.Join(tmp, "taito")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "installed.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := install.LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if len(loaded.Installed.Skills) != 1 {
		t.Fatalf("expected 1 skill entry, got %d", len(loaded.Installed.Skills))
	}
	if loaded.Installed.Skills[0].Version != "3.2.1" {
		t.Errorf("Version = %q, want %q", loaded.Installed.Skills[0].Version, "3.2.1")
	}
}

func TestListEmptyVersionGraceful(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	idx := &install.InstalledIndex{
		Version: install.IndexSpecVersion,
		Installed: install.InstalledPackages{
			Skills: []install.InstalledEntry{
				{
					Name:      "old-skill",
					SpecType:  "skill",
					Reference: "ghcr.io/org/old:1.0.0",
					InstallIn: []install.InstallLocation{{Tool: "cursor", Path: "/home/user/.cursor/skills/old-skill"}},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(idx, "", "  ")
	configDir := filepath.Join(tmp, "taito")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "installed.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := install.LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if len(loaded.Installed.Skills) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Installed.Skills))
	}
	if loaded.Installed.Skills[0].Version != "" {
		t.Errorf("Version = %q, want empty", loaded.Installed.Skills[0].Version)
	}
}
