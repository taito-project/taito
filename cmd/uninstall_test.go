package cmd

import (
	"testing"
)

func TestUninstallCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "uninstall" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'uninstall' command to be registered on rootCmd")
	}
}

func TestUninstallCmdHasAlias(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "uninstall" {
			aliases := c.Aliases
			found := false
			for _, a := range aliases {
				if a == "rm" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected 'rm' alias, got aliases: %v", aliases)
			}
			return
		}
	}
	t.Error("uninstall command not found")
}

func TestUninstallCmdRequiresExactlyOneArg(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Name() == "uninstall" {
			// cobra.ExactArgs(1) should be set — verify by checking Args is non-nil.
			if c.Args == nil {
				t.Error("expected Args validation to be set (ExactArgs(1))")
			}
			return
		}
	}
	t.Error("uninstall command not found")
}
