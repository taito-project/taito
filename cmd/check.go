package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/taito-project/taito/internal/spec"
	"github.com/taito-project/taito/ui"
)

var checkPath string

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Validate a taito.spec manifest file",
	Long: `Checks whether a taito.spec file is valid according to the v0.1.0 specification.

By default, looks for a taito.spec file in the current directory. Pass a path as
an argument or use --path to point to a specific file or directory.

Examples:
  taito check
  taito check ./skills/git-helper/taito.spec
  taito check ./skills/git-helper
  taito check --path ./skills/git-helper/taito.spec

Hard errors (missing type, missing name, invalid type) cause the check to fail.
All other issues are reported as warnings.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Positional argument takes precedence over --path default.
		if len(args) > 0 {
			checkPath = args[0]
		}

		// Determine whether the path is a file or directory.
		info, err := os.Stat(checkPath)
		if err != nil {
			fmt.Printf("Error: cannot access path '%s': %v\n", checkPath, err)
			os.Exit(1)
		}

		displayPath := checkPath

		checkFn := func() tea.Msg {
			var s *spec.TaitoSpec
			var loadErr error

			if info.IsDir() {
				s, loadErr = spec.LoadFromDir(checkPath)
			} else {
				s, loadErr = spec.Load(checkPath)
			}

			if loadErr != nil {
				return ui.CheckResultMsg{Err: loadErr}
			}

			warnings, valErr := spec.Validate(s)
			if valErr != nil {
				return ui.CheckResultMsg{Err: valErr}
			}

			warnStrings := make([]string, len(warnings))
			for i, w := range warnings {
				warnStrings[i] = fmt.Sprintf("%s: %s", w.Field, w.Message)
			}

			return ui.CheckResultMsg{
				Warnings: warnStrings,
				SpecName: s.Name,
				SpecType: s.Type,
			}
		}

		m := ui.NewCheckModel(displayPath, checkFn)
		p := tea.NewProgram(m)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running tea program: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringVar(&checkPath, "path", ".", "Path to a taito.spec file or directory containing one")
}
