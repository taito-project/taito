package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/ui"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <id>",
	Short: "Uninstall a package by ID from all configured tools",
	Long: `Uninstall a skill, agent, or bundle by its ID from all configured AI coding tools.

The package is removed from every tool it was installed to. If the package is
a bundle, all child skills and agents that were installed by that bundle are
also removed.

Examples:
  taito uninstall ffe390a1
  taito rm 9402ea07`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	Run:     runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) {
	id := args[0]

	lookupFn := func() tea.Msg {
		result, err := install.LookupByID(id)
		if err != nil {
			return ui.UninstallLookupMsg{Err: err}
		}
		return ui.UninstallLookupMsg{
			Name:  result.Name,
			Count: result.Count,
		}
	}

	uninstallFn := func() tea.Msg {
		results, err := install.Uninstall(id)
		if err != nil {
			return ui.UninstallResultMsg{Err: err}
		}

		if len(results) == 0 {
			return ui.UninstallResultMsg{
				Err: fmt.Errorf("no entries were removed"),
			}
		}

		specType, name, toolResults := install.GroupResultsByTool(results)

		var tools []ui.UninstallToolResult
		for _, tr := range toolResults {
			tools = append(tools, ui.UninstallToolResult{
				Tool:  tr.Tool,
				Paths: tr.Paths,
			})
		}

		return ui.UninstallResultMsg{
			Name:       name,
			SpecType:   specType,
			Tools:      tools,
			TotalCount: len(results),
		}
	}

	m := ui.NewUninstallModel(id, lookupFn, uninstallFn)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Uninstall failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	if um, ok := finalModel.(ui.UninstallModel); ok && um.Err() != nil {
		os.Exit(1)
	}
}
