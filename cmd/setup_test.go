package cmd

import (
	"testing"

	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/ui"
)

func TestSetupCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "setup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("setup command not registered on rootCmd")
	}
}

func TestSetupCommandMetadata(t *testing.T) {
	if setupCmd.Use != "setup" {
		t.Errorf("Use = %q, want %q", setupCmd.Use, "setup")
	}
	if setupCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestSetupModelInitialState(t *testing.T) {
	m := ui.NewSetupModel(nil)

	if m.Cancelled() {
		t.Error("new model should not be cancelled")
	}
	if m.Result() != nil {
		t.Error("new model should have nil result")
	}
	if m.Err() != nil {
		t.Errorf("new model should have nil error, got %v", m.Err())
	}
}

func TestSetupModelPreFill(t *testing.T) {
	existing := &config.Config{
		Storage: config.StorageConfig{CacheDir: "/custom/cache"},
		Tools: []config.ToolConfig{
			{Name: "cursor"},
			{Name: "opencode"},
		},
	}

	m := ui.NewSetupModel(existing)

	// Model should be created without error.
	if m.Cancelled() {
		t.Error("pre-filled model should not be cancelled")
	}
	if m.Result() != nil {
		t.Error("pre-filled model should not have a result yet")
	}
}

func TestSetupModelViewNotEmpty(t *testing.T) {
	m := ui.NewSetupModel(nil)
	v := m.View()
	if v == "" {
		t.Error("View() should not return empty string")
	}
	if len(v) < 20 {
		t.Errorf("View() seems too short: %q", v)
	}
}

func TestSetupModelViewContainsTitle(t *testing.T) {
	m := ui.NewSetupModel(nil)
	v := m.View()
	// The title should contain "Taito Setup" (possibly with ANSI codes).
	// Check for the raw text.
	if !containsAny(v, "Taito Setup", "taito setup") {
		t.Errorf("View() should contain 'Taito Setup', got: %s", v)
	}
}

func TestSetupModelViewContainsToolPrompt(t *testing.T) {
	m := ui.NewSetupModel(nil)
	v := m.View()
	if !containsAny(v, "AI coding tools", "tools you use") {
		t.Errorf("View() should contain tool selection prompt, got: %s", v)
	}
}

func TestSetupCommandHasToolsFlag(t *testing.T) {
	f := setupCmd.Flags().Lookup("tools")
	if f == nil {
		t.Fatal("expected --tools flag to be registered")
	}
	if f.DefValue != "" {
		t.Errorf("expected default value to be empty, got %q", f.DefValue)
	}
}

// This replaces the old global Cfg variable.
func testConfig() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		return &config.Config{}
	}
	return cfg
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
