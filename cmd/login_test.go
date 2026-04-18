package cmd

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/taito-project/taito/ui"
)

func TestLoginCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "login" {
			found = true
			break
		}
	}
	if !found {
		t.Error("login command not registered on rootCmd")
	}
}

func TestLoginCommandMetadata(t *testing.T) {
	if loginCmd.Use != "login <registry>" {
		t.Errorf("Use = %q, want %q", loginCmd.Use, "login <registry>")
	}
	if loginCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestLoginCommandRequiresArg(t *testing.T) {
	// cobra.ExactArgs(1) should be set.
	if loginCmd.Args == nil {
		t.Error("Args should not be nil (expects ExactArgs(1))")
	}
}

func TestLoginCommandHasFlags(t *testing.T) {
	f := loginCmd.Flags()

	if f.Lookup("username") == nil {
		t.Error("missing --username flag")
	}
	if f.Lookup("password") == nil {
		t.Error("missing --password flag")
	}
	if f.Lookup("password-stdin") == nil {
		t.Error("missing --password-stdin flag")
	}

	// Check shorthand flags.
	if f.ShorthandLookup("u") == nil {
		t.Error("missing -u shorthand for --username")
	}
	if f.ShorthandLookup("p") == nil {
		t.Error("missing -p shorthand for --password")
	}
}

func TestLoginModelInitialState(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel("ghcr.io", "", noopLogin)

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

func TestLoginModelPrefilledUsername(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel("ghcr.io", "myuser", noopLogin)

	if m.Cancelled() {
		t.Error("pre-filled model should not be cancelled")
	}
	if m.Result() != nil {
		t.Error("pre-filled model should not have result yet")
	}
}

func TestLoginModelViewNotEmpty(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel("ghcr.io", "", noopLogin)
	v := m.View()
	if v == "" {
		t.Error("View() should not return empty string")
	}
	if len(v) < 20 {
		t.Errorf("View() seems too short: %q", v)
	}
}

func TestLoginModelViewContainsTitle(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel("ghcr.io", "", noopLogin)
	v := m.View()
	if !containsAny(v, "taito login") {
		t.Errorf("View() should contain 'taito login', got: %s", v)
	}
}

func TestLoginModelViewContainsRegistry(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel("ghcr.io", "", noopLogin)
	v := m.View()
	if !containsAny(v, "ghcr.io") {
		t.Errorf("View() should contain registry name, got: %s", v)
	}
}

func TestLoginModelViewPasswordStep(t *testing.T) {
	noopLogin := func(reg, user, pass string) tea.Msg {
		return ui.LoginResultMsg{}
	}

	// When username is pre-filled, should start at password step.
	m := ui.NewLoginModel("ghcr.io", "myuser", noopLogin)
	v := m.View()
	if !containsAny(v, "Password", "password") {
		t.Errorf("View() with pre-filled username should show password prompt, got: %s", v)
	}
}
