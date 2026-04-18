package cmd

import (
	"testing"
)

func TestCheckCmdExists(t *testing.T) {
	// Verify the check command is registered as a subcommand of root.
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'check' command to be registered on rootCmd")
	}
}

func TestCheckCmdPathFlagDefault(t *testing.T) {
	checkCmd.ResetFlags()
	checkCmd.Flags().StringVar(&checkPath, "path", ".", "Path to a taito.spec file or directory containing one")

	val, err := checkCmd.Flags().GetString("path")
	if err != nil {
		t.Fatalf("Failed getting path flag: %v", err)
	}
	if val != "." {
		t.Errorf("Expected default path '.', got '%s'", val)
	}
}

func TestCheckCmdPathFlagSet(t *testing.T) {
	checkCmd.ResetFlags()
	checkCmd.Flags().StringVar(&checkPath, "path", ".", "Path to a taito.spec file or directory containing one")

	if err := checkCmd.Flags().Set("path", "./my-skill/taito.spec"); err != nil {
		t.Fatalf("Failed setting path flag: %v", err)
	}

	val, _ := checkCmd.Flags().GetString("path")
	if val != "./my-skill/taito.spec" {
		t.Errorf("Expected path './my-skill/taito.spec', got '%s'", val)
	}
}

func TestCheckCmdPathFlagDirectory(t *testing.T) {
	checkCmd.ResetFlags()
	checkCmd.Flags().StringVar(&checkPath, "path", ".", "Path to a taito.spec file or directory containing one")

	if err := checkCmd.Flags().Set("path", "./skills/git-helper"); err != nil {
		t.Fatalf("Failed setting path flag to directory: %v", err)
	}

	val, _ := checkCmd.Flags().GetString("path")
	if val != "./skills/git-helper" {
		t.Errorf("Expected path './skills/git-helper', got '%s'", val)
	}
}

func TestCheckCmdPathNotRequired(t *testing.T) {
	checkCmd.ResetFlags()
	checkCmd.Flags().StringVar(&checkPath, "path", ".", "Path to a taito.spec file or directory containing one")

	// The path flag should NOT be marked as required (it has a default).
	f := checkCmd.Flags().Lookup("path")
	if f == nil {
		t.Fatal("path flag not found")
	}
	annotations := f.Annotations
	if annotations != nil {
		if vals, ok := annotations["cobra_annotation_bash_completion_one_required_flag"]; ok {
			if len(vals) > 0 && vals[0] == "true" {
				t.Error("path flag should not be required (has default value)")
			}
		}
	}
}
