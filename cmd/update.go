package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/internal/update"
	"github.com/taito-project/taito/ui"
)

var updateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Check for and apply updates to installed packages",
	Long: `Check for newer versions of installed packages and update them.

Without arguments, all installed packages are checked. When an ID is
provided, only that specific package is checked.

For OCI packages, tags are listed from the registry and compared using
semantic versioning. For GitHub packages, tags are fetched from the GitHub
API and compared similarly; if the installed version is a commit SHA, the
latest commit is compared.

After checking, a summary is displayed and you are asked to confirm
before any updates are applied.

Examples:
  taito update
  taito update ffe390a1`,
	Aliases: []string{"up"},
	Args:    cobra.MaximumNArgs(1),
	Run:     runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	cfg := cfgFromCmd(cmd)

	if len(cfg.Tools) == 0 {
		fmt.Printf(" %s  no tools configured — run 'taito setup' first\n", ui.FailIcon())
		os.Exit(1)
	}

	var filterID string
	if len(args) > 0 {
		filterID = args[0]
	}

	check := update.CheckAll
	if filterID != "" {
		check = func() ([]update.UpdateResult, error) {
			return update.CheckByID(filterID)
		}
	}

	// checkFn runs the version check in a background goroutine.
	checkFn := func() tea.Msg {
		results, err := check()
		if err != nil {
			return ui.UpdateCheckMsg{Err: err}
		}

		return ui.UpdateCheckMsg{Entries: toUpdateEntries(results)}
	}

	installFn := func(ref string) tea.Msg {
		result := update.Install(ref, cfg)
		return ui.UpdateInstallMsg{Name: result.Name, Err: result.Err}
	}

	m := ui.NewUpdateModel(filterID, checkFn, installFn)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Update failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	if um, ok := finalModel.(ui.UpdateModel); ok && um.Err() != nil {
		fmt.Printf(" %s  %v\n", ui.FailIcon(), um.Err())
		os.Exit(1)
	}
}

func toUpdateEntries(results []update.UpdateResult) []ui.UpdateEntry {
	entries := make([]ui.UpdateEntry, len(results))
	for i, r := range results {
		entries[i] = ui.UpdateEntry{
			ID:             r.ID,
			Name:           r.Name,
			SpecType:       r.SpecType,
			CurrentVersion: r.CurrentVersion,
			LatestVersion:  r.LatestVersion,
			Reference:      r.Reference,
			UpdateRef:      r.UpdateReference,
			HasUpdate:      r.HasUpdate,
			IsLocal:        r.IsLocal,
			Error:          r.Error,
			IsBundleChild:  r.IsBundleChild,
			BundleID:       r.BundleID,
		}
	}

	return entries
}
