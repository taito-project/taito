package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/ui"
)

// configKey is the context key for the loaded user configuration.
type configKey struct{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "taito",
	Short: "A beautiful and powerful CLI for Taito",
	Long:  `taito is a terminal application install and update agent skills and bundles`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load user configuration from the platform-appropriate config directory.
		// A missing config file is fine — defaults are used.
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
			cfg = &config.Config{}
		}
		cmd.SetContext(context.WithValue(cmd.Context(), configKey{}, cfg))
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Launch the Bubble Tea interface when no subcommands are passed
		p := tea.NewProgram(ui.InitialModel())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

// cfgFromCmd retrieves the loaded configuration from the command context.
func cfgFromCmd(cmd *cobra.Command) *config.Config {
	if cfg, ok := cmd.Context().Value(configKey{}).(*config.Config); ok {
		return cfg
	}
	return &config.Config{}
}

// setCmdConfig stores a configuration in the command context. This is used
// by the setup command after saving a new configuration, and by tests.
func setCmdConfig(cmd *cobra.Command, cfg *config.Config) {
	cmd.SetContext(context.WithValue(cmd.Context(), configKey{}, cfg))
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
