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

Use --tools to skip the interactive prompt (useful for CI/automation):
  taito setup --tools=opencode,claude-code

Examples:
  taito setup
  taito setup --tools=opencode,claude-code`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := cfgFromCmd(cmd)

		toolsFlag, _ := cmd.Flags().GetString("tools")
		if toolsFlag != "" {
			runSetupWithTools(cmd, cfg, toolsFlag)
			return
		}

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

		saveSetupConfig(cmd, cfg, result.Tools)
		printConfigPath()
	},
}

func init() {
	setupCmd.Flags().String("tools", "", "Comma-separated list of tools to configure (skips interactive prompt)")
	rootCmd.AddCommand(setupCmd)
}

// runSetupWithTools handles the non-interactive path when --tools is provided.
func runSetupWithTools(cmd *cobra.Command, cfg *config.Config, toolsFlag string) {
	tools, err := config.ResolveTools(toolsFlag)
	if err != nil {
		fmt.Printf(" %s  %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	saveSetupConfig(cmd, cfg, tools)

	fmt.Printf(" %s  Configuration saved\n", ui.SuccessIcon())
	for _, tc := range tools {
		fmt.Printf("    %s\n", tc.Name)
	}
	printConfigPath()
}

// saveSetupConfig persists the tool configuration to disk.
func saveSetupConfig(cmd *cobra.Command, existing *config.Config, tools []config.ToolConfig) {
	newCfg := &config.Config{
		Storage: existing.Storage,
		Tools:   tools,
	}

	if err := config.Save(newCfg); err != nil {
		fmt.Printf(" %s  Failed to save configuration\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	setCmdConfig(cmd, newCfg)
}

// printConfigPath prints the config file location.
func printConfigPath() {
	cfgPath, err := config.ConfigFilePath()
	if err == nil {
		fmt.Printf("\n    Saved to %s\n\n", cfgPath)
	}
}
