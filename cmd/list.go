package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/ui"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed packages",
	Long: `List all skills, agents, and bundles currently installed across your
configured AI coding tools.

The output shows each installed package with its name, type, tool, version,
and source reference. Packages installed via a bundle are grouped under
their parent bundle in a tree structure.

Examples:
  taito list`,
	Aliases: []string{"ls"},
	Run:     runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	idx, err := install.LoadInstalled()
	if err != nil {
		fmt.Printf(" %s  Failed to load installed packages\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	if len(idx.Installed.Skills) == 0 && len(idx.Installed.Agents) == 0 && len(idx.Installed.Bundles) == 0 {
		fmt.Printf(" %s  No packages installed\n", ui.SuccessIcon())
		return
	}

	toolDisplayNames := make(map[string]string)
	for _, kt := range config.KnownTools() {
		toolDisplayNames[kt.Name] = kt.DisplayName
	}

	rows := install.BuildTreeRows(idx.Installed, toolDisplayNames)

	headerStyle := lipgloss.NewStyle().
		Foreground(ui.ColorPrimary).
		Bold(true).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderRow(false).
		BorderColumn(false).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderHeader(false).
		Headers("ID", "TAITO", "TYPE", "TOOL", "VERSION").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	fmt.Println(t)
}
