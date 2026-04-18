package cmd

import (
	"testing"
)

func TestLogoutCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "logout" {
			found = true
			break
		}
	}
	if !found {
		t.Error("logout command not registered on rootCmd")
	}
}

func TestLogoutCommandMetadata(t *testing.T) {
	if logoutCmd.Use != "logout <registry>" {
		t.Errorf("Use = %q, want %q", logoutCmd.Use, "logout <registry>")
	}
	if logoutCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestLogoutCommandRequiresArg(t *testing.T) {
	if logoutCmd.Args == nil {
		t.Error("Args should not be nil (expects ExactArgs(1))")
	}
}

func TestLogoutCommandNoExtraFlags(t *testing.T) {
	// Logout is simple — it should not have custom flags
	// beyond the inherited ones from root.
	f := logoutCmd.Flags()
	if f.Lookup("username") != nil {
		t.Error("logout should not have --username flag")
	}
	if f.Lookup("password") != nil {
		t.Error("logout should not have --password flag")
	}
}
