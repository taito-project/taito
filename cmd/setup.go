package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/ui"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure the taito CLI",
	Long: `Interactive setup wizard for configuring the taito CLI.

Prompts for:
  - AI coding tools (Copilot, Claude Code, OpenCode)

Run again at any time to update your configuration.

Examples:
  taito setup`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgFromCmd(cmd)
		m := ui.NewSetupModel(cfg)

		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf(" %s  Setup failed\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}

		final, ok := finalModel.(ui.SetupModel)
		if !ok {
			fmt.Printf(" %s  Setup failed\n    unexpected model type\n", ui.FailIcon())
			os.Exit(1)
		}

		if final.Cancelled() {
			return
		}

		result := final.Result()
		if result == nil {
			return
		}

		// Build the config from the wizard result.
		// Preserve existing storage settings (setup only manages tools).
		newCfg := &config.Config{
			Storage: cfg.Storage,
			Tools:   result.Tools,
		}

		// Save the config to disk.
		if err := config.Save(newCfg); err != nil {
			fmt.Printf(" %s  Failed to save configuration\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}

		// Update the in-memory config so subsequent commands (if any) see it.
		setCmdConfig(cmd, newCfg)

		// The done view was already printed by Bubble Tea.
		// Append the config file path.
		cfgPath, err := config.ConfigFilePath()
		if err == nil {
			fmt.Printf("    Saved to %s\n\n", cfgPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
