package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/taito-project/taito/internal/cache"
	"github.com/taito-project/taito/ui"
)

var (
	specFlag   string
	formatFlag string
	outputFlag string
)

var validFormats = map[string]bool{
	"tar.gz": true,
	"oci":    true,
}

var packageCmd = &cobra.Command{
	Use:   "package [reference]",
	Short: "Package a skill, agent, or bundle into an OCI artifact",
	Long: `Packages a skill, agent, or bundle into an OCI artifact for publication
and distribution. The taito.spec file in the context directory is loaded and
validated automatically.

The optional positional argument is an OCI reference (e.g. registry/name:tag).
If omitted, a reference is derived from the spec name and version.

By default, artifacts are stored in the taito cache directory
(e.g. ~/.cache/taito/packages/). Use --output to write to a custom path.

Examples:
  taito package
  taito package myregistry.io/coolskill:latest
  taito package myregistry.io/coolskill:latest --spec=./my-skill
  taito package --spec=./skills/git-helper --format=tar.gz
  taito package --output=./dist

Supported formats:
  oci     (default) — an OCI Image Layout directory
  tar.gz  — a compressed tar archive`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Validate format.
		if !validFormats[formatFlag] {
			fmt.Printf("Error: unsupported format %q. Valid formats: tar.gz, oci\n", formatFlag)
			os.Exit(1)
		}

		// Resolve the spec context directory.
		specPath := specFlag
		info, err := os.Stat(specPath)
		if err != nil {
			fmt.Printf("Error: cannot access spec path '%s': %v\n", specPath, err)
			os.Exit(1)
		}

		// Determine the context directory and spec file path.
		var contextDir string
		var specFile string
		if info.IsDir() {
			contextDir = specPath
			specFile = filepath.Join(specPath, "taito.spec")
		} else {
			contextDir = filepath.Dir(specPath)
			specFile = specPath
		}

		// Capture the user-provided reference (if any).
		var userRef string
		if len(args) > 0 {
			userRef = args[0]
		}

		output := outputFlag

		// Build a resolveTarget closure that the model uses to determine
		// where to write the output artifact.
		resolveTarget := func(reference, format string) (string, error) {
			if output != "" {
				return output, nil
			}

			cacheDir, err := cfgFromCmd(cmd).CacheDir()
			if err != nil {
				return "", fmt.Errorf("cannot determine cache directory: %w", err)
			}

			packagesDir := filepath.Join(cacheDir, "packages")
			if err := os.MkdirAll(packagesDir, 0755); err != nil {
				return "", fmt.Errorf("cannot create packages directory: %w", err)
			}

			return cache.PackagePath(cacheDir, reference), nil
		}

		m := ui.NewPackageModel(specFile, contextDir, formatFlag, userRef, resolveTarget)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error running tea program: %v\n", err)
			os.Exit(1)
		}
		if pm, ok := finalModel.(ui.PackageModel); ok && pm.Err() != nil {
			os.Exit(1)
		}

		// On success, update the cache index.
		if pm, ok := finalModel.(ui.PackageModel); ok && pm.Err() == nil {
			cacheDir, cacheErr := cfgFromCmd(cmd).CacheDir()
			if cacheErr == nil && output == "" {
				// Only update index when storing in the cache directory.
				if err := cache.AddEntry(cacheDir, pm.Reference(), pm.SpecType(), formatFlag); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to update cache index: %v\n", err)
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(packageCmd)

	packageCmd.Flags().StringVar(&specFlag, "spec", ".", "Path to a taito.spec file or directory containing one")
	packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format: oci (default) or tar.gz")
	packageCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output path (default: cache directory)")
}
