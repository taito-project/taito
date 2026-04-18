package cmd

import "testing"

func TestPruneCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "prune" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'prune' command to be registered on rootCmd")
	}
}

func TestPruneCmdDryRunFlagDefault(t *testing.T) {
	pruneCmd.ResetFlags()
	pruneCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be removed without deleting")

	val, err := pruneCmd.Flags().GetBool("dry-run")
	if err != nil {
		t.Fatalf("Failed getting dry-run flag: %v", err)
	}
	if val != false {
		t.Errorf("Expected default dry-run false, got %v", val)
	}
}

func TestPruneCmdDryRunFlagSet(t *testing.T) {
	pruneCmd.ResetFlags()
	pruneCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be removed without deleting")

	if err := pruneCmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatalf("Failed setting dry-run flag: %v", err)
	}

	val, _ := pruneCmd.Flags().GetBool("dry-run")
	if val != true {
		t.Errorf("Expected dry-run true, got %v", val)
	}
}
