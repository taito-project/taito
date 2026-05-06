package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/internal/github"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/oci"
	"github.com/taito-project/taito/ui"
)

var installCmd = &cobra.Command{
	Use:   "install <reference|path>",
	Short: "Install a skill, agent, or bundle to configured tools",
	Long: `Install a taito artifact from an OCI registry, a local OCI layout, or a
GitHub repository into all configured AI coding tools (Cursor, Claude Code, etc.).

The source can be:
  - An OCI registry reference (e.g. ghcr.io/org/my-skill:1.0.0)
  - A local OCI layout directory (must contain an oci-layout file)
  - A GitHub repository (e.g. github.com/anthropics/skills@0.0.1) 
  
Note that GitHub repositories must be public, as authentication is not currently supported.

Bundles are expanded: each child skill/agent in the bundle's includes is
installed to its own directory.

You must run 'taito setup' first to configure your tools.

Examples:
  taito install ghcr.io/org/my-skill:1.0.0
  taito install ./local-oci-layout
  taito install github.com/anthropics/skills@0.0.1
  taito install github.com/anthropics/skills/agents/devops@main`,
	Args: cobra.ExactArgs(1),
	Run:  runInstall,
}

func init() {
	installCmd.Flags().Bool("non-interactive", false, "Install all items from bundles without prompting (useful for CI/automation)")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) {
	source := args[0]
	cfg := cfgFromCmd(cmd)

	if len(cfg.Tools) == 0 {
		fmt.Printf(" %s  no tools configured — run 'taito setup' first\n", ui.FailIcon())
		os.Exit(1)
	}

	var installer ui.Installer
	if install.IsLocalPath(source) || !github.IsGitHubSource(source) {
		installer = oci.NewInstaller(source, cfg)
	} else {
		installer = github.NewInstaller(source, cfg)
	}

	if cleaner, ok := installer.(interface{ Cleanup() }); ok {
		defer cleaner.Cleanup()
	}

	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")

	m := ui.NewInstallModel(source, installer, nonInteractive)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		if cleaner, ok := installer.(interface{ Cleanup() }); ok {
			cleaner.Cleanup()
		}
		fmt.Printf(" %s  Install failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	if im, ok := finalModel.(ui.InstallModel); ok && im.Err() != nil {
		if cleaner, ok := installer.(interface{ Cleanup() }); ok {
			cleaner.Cleanup()
		}
		os.Exit(1)
	}
}
