package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new taito.spec file",
	Long: `Initialize a new taito.spec file interactively.

This command launches a wizard that helps you generate a valid taito.spec
manifest file in the current directory for a skill, agent, or bundle.

Examples:
  taito init`,
	Args: cobra.NoArgs,
	Run:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
	if _, err := os.Stat("taito.spec"); err == nil {
		fmt.Printf(" %s  taito.spec already exists in this directory\n", ui.WarningIcon())
		os.Exit(1)
	}

	m := ui.NewInitModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Error running init wizard: %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	im, ok := finalModel.(ui.InitModel)
	if !ok || im.Cancelled() || im.Result() == nil {
		fmt.Printf("\n %s  Init cancelled\n", ui.WarningIcon())
		return
	}

	spec := im.Result().Spec
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Printf(" %s  Error formatting spec: %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	if err := os.WriteFile("taito.spec", data, 0644); err != nil {
		fmt.Printf(" %s  Error writing taito.spec: %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	fmt.Printf(" %s  Successfully created taito.spec for %s '%s'\n", ui.SuccessIcon(), spec.Type, spec.Name)
}
