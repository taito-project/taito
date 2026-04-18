package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/content/oci"

	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/cache"
	"github.com/taito-project/taito/internal/registry"
	"github.com/taito-project/taito/ui"
)

var (
	pushSource string
)

var pushCmd = &cobra.Command{
	Use:   "push <reference>",
	Short: "Push a packaged artifact to an OCI registry",
	Long: `Push a local OCI layout (created by taito package) to a remote OCI registry.

If --source is not specified, the artifact is located in the taito cache
directory based on the reference.

You must be logged in first (see taito login).

Examples:
  taito push ghcr.io/org/my-skill:1.0.0
  taito push ghcr.io/org/my-skill:1.0.0 --source ./dist/my-skill-oci`,
	Args: cobra.ExactArgs(1),
	Run:  runPush,
}

func init() {
	pushCmd.Flags().StringVarP(&pushSource, "source", "s", "", "Source OCI layout path (default: cache directory)")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) {
	reference := args[0]

	// Resolve source path.
	sourcePath := pushSource
	if sourcePath == "" {
		cacheDir, err := cfgFromCmd(cmd).CacheDir()
		if err != nil {
			fmt.Printf(" %s  %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}
		sourcePath = cache.PackagePath(cacheDir, reference)
	}

	// Verify source exists.
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		fmt.Printf(" %s  Source OCI layout not found: %s\n", ui.FailIcon(), sourcePath)
		fmt.Printf("    Run 'taito package' first to create the artifact.\n")
		os.Exit(1)
	}

	// Read spec type from the local manifest annotations for the type badge.
	specType := readLocalSpecType(sourcePath, reference)

	resolveFn := func() tea.Msg {
		// Validate that the local layout is a valid OCI store.
		_, err := oci.New(sourcePath)
		if err != nil {
			return ui.PushResolveMsg{Err: fmt.Errorf("invalid OCI layout at %s: %w", sourcePath, err)}
		}
		return ui.PushResolveMsg{}
	}

	pushFn := func() tea.Msg {
		ctx := context.Background()
		desc, err := registry.Push(ctx, sourcePath, reference)
		if err != nil {
			return ui.PushResultMsg{Err: err}
		}
		return ui.PushResultMsg{
			Digest:   desc.Digest.String(),
			SpecType: specType,
		}
	}

	m := ui.NewPushModel(reference, resolveFn, pushFn)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Push failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}
	if pm, ok := finalModel.(ui.PushModel); ok && pm.Err() != nil {
		os.Exit(1)
	}
}

// readLocalSpecType reads the dev.taito.spec.type annotation from the local
// OCI layout manifest. Returns empty string on any error.
func readLocalSpecType(ociPath, reference string) string {
	store, err := oci.New(ociPath)
	if err != nil {
		return ""
	}

	ctx := context.Background()
	tag := registry.TagFromReference(reference)

	desc, err := store.Resolve(ctx, tag)
	if err != nil {
		return ""
	}

	rc, err := store.Fetch(ctx, desc)
	if err != nil {
		return ""
	}
	defer func() { _ = rc.Close() }()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return ""
	}

	if manifest.Annotations != nil {
		return manifest.Annotations[archive.AnnotationSpecType]
	}
	return ""
}
