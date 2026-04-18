package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/taito-project/taito/internal/cache"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/registry"
	"github.com/taito-project/taito/ui"
)

var (
	pullOutput string
)

var pullCmd = &cobra.Command{
	Use:   "pull <reference>",
	Short: "Pull an artifact from an OCI registry",
	Long: `Pull a taito artifact from a remote OCI registry into a local OCI layout.

By default, the artifact is stored in the taito cache directory
(e.g. ~/.cache/taito/packages/). Use --output to write to a custom path.

The pulled artifact is validated before being committed to disk. If validation
fails (wrong artifact type, missing spec type annotation, or unknown spec type),
the artifact is discarded.

You must be logged in first (see taito login).

Examples:
  taito pull ghcr.io/org/my-skill:1.0.0
  taito pull ghcr.io/org/my-skill:1.0.0 --output ./local-skills/my-skill`,
	Args: cobra.ExactArgs(1),
	Run:  runPull,
}

func init() {
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output OCI layout path (default: cache directory)")
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) {
	reference := args[0]

	targetPath, usingCache, err := resolveTargetPath(pullOutput, reference, cfgFromCmd(cmd))
	if err != nil {
		fmt.Printf(" %s  %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	resolveFn := func() tea.Msg {
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return ui.PullResolveMsg{Err: fmt.Errorf("cannot create output directory: %w", err)}
		}
		return ui.PullResolveMsg{}
	}

	pullFn := func() tea.Msg {
		ctx := context.Background()
		desc, specType, warnings, err := registry.Pull(ctx, reference, targetPath)
		if err != nil {
			return ui.PullCopyMsg{Err: err}
		}
		return ui.PullCopyMsg{
			Digest:   desc.Digest.String(),
			SpecType: specType,
			Warnings: warnings,
		}
	}

	m := ui.NewPullModel(reference, resolveFn, pullFn)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Pull failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	if pm, ok := finalModel.(ui.PullModel); ok && pm.Err() != nil {
		os.Exit(1)
	}

	if usingCache {
		updatePullCacheIndex(finalModel, reference, cfgFromCmd(cmd))
	}
}

// resolveTargetPath determines the pull target path from the --output flag
// or the default cache directory.
func resolveTargetPath(output, reference string, cfg *config.Config) (targetPath string, usingCache bool, err error) {
	if output != "" {
		return output, false, nil
	}

	cacheDir, err := cfg.CacheDir()
	if err != nil {
		return "", false, err
	}

	packagesDir := filepath.Join(cacheDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return "", false, fmt.Errorf("cannot create packages directory: %w", err)
	}

	return cache.PackagePath(cacheDir, reference), true, nil
}

// updatePullCacheIndex adds a cache entry on successful pull.
func updatePullCacheIndex(finalModel tea.Model, reference string, cfg *config.Config) {
	pm, ok := finalModel.(ui.PullModel)
	if !ok || pm.Err() != nil {
		return
	}
	cacheDir, err := cfg.CacheDir()
	if err != nil {
		return
	}
	if err := cache.AddEntry(cacheDir, reference, pm.SpecType(), "oci"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update cache index: %v\n", err)
	}
}
