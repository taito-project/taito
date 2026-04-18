package oci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/taito-project/taito/internal/cache"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/registry"
	"github.com/taito-project/taito/internal/spec"
	"github.com/taito-project/taito/ui"
)

// Installer implements ui.Installer for OCI registry and local layout sources.
type Installer struct {
	source     string
	cfg        *config.Config
	layoutPath string
	spec       *spec.TaitoSpec
}

func NewInstaller(source string, cfg *config.Config) *Installer {
	return &Installer{
		source: source,
		cfg:    cfg,
	}
}

func (i *Installer) Resolve() tea.Msg {
	// Check if it's a local path.
	if install.IsLocalPath(i.source) {
		if !install.IsOCILayout(i.source) {
			return ui.InstallResolveMsg{Err: fmt.Errorf("not a valid OCI layout at %s", i.source)}
		}
		i.layoutPath = i.source
		return ui.InstallResolveMsg{}
	}

	// Registry reference — check cache first.
	cacheDir, err := i.cfg.CacheDir()
	if err != nil {
		return ui.InstallResolveMsg{Err: fmt.Errorf("cache directory: %w", err)}
	}

	packagesDir := filepath.Join(cacheDir, "packages")
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		return ui.InstallResolveMsg{Err: fmt.Errorf("create packages directory: %w", err)}
	}

	cachedPath := cache.PackagePath(cacheDir, i.source)
	if install.IsOCILayout(cachedPath) {
		i.layoutPath = cachedPath
		return ui.InstallResolveMsg{}
	}

	// Not cached — pull from registry.
	ctx := context.Background()
	_, specType, _, err := registry.Pull(ctx, i.source, cachedPath)
	if err != nil {
		return ui.InstallResolveMsg{Err: fmt.Errorf("pull: %w", err)}
	}

	// Update cache index.
	if err := cache.AddEntry(cacheDir, i.source, specType, "oci"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update cache index: %v\n", err)
	}

	i.layoutPath = cachedPath
	return ui.InstallResolveMsg{}
}

func (i *Installer) Extract() tea.Msg {
	s, err := install.ReadSpecFromLayout(i.layoutPath)
	if err != nil {
		return ui.InstallExtractMsg{Err: err}
	}
	i.spec = s

	var items []ui.SelectableItem
	if s.Type == spec.TypeBundle {
		items = resolveBundleItems(i.layoutPath, s.Includes)
	}

	return ui.InstallExtractMsg{
		Name:     s.Name,
		SpecType: s.Type,
		Items:    items,
		OnSelect: func(chosenIDs []string) error {
			i.spec.Includes = chosenIDs
			return nil
		},
	}
}

// resolveBundleItems reads child specs from the OCI layout's layer blob and
// returns selectable items for bundle skill/agent selection.
func resolveBundleItems(layoutPath string, includes []string) []ui.SelectableItem {
	layerPath, err := install.ResolveLayerBlobPath(layoutPath)
	if err != nil {
		return nil
	}

	archRoot, err := install.DiscoverArchiveRoot(layerPath)
	if err != nil {
		return nil
	}

	var items []ui.SelectableItem
	for _, inc := range includes {
		clean := filepath.Clean(inc)
		childSpecRel := strings.TrimPrefix(clean, "./")
		archSpecPath := filepath.ToSlash(filepath.Join(archRoot, childSpecRel))

		childSpec, err := install.ReadSpecEntryFromTarGz(layerPath, archSpecPath)
		if err != nil {
			continue
		}

		items = append(items, ui.SelectableItem{
			ID:       inc,
			Name:     childSpec.Name,
			SpecType: childSpec.Type,
		})
	}
	return items
}

func (i *Installer) Install() tea.Msg {
	// For registry sources, override the spec version with the tag used
	// during install (e.g. "v1.0.0" from "registry.example.com/skill:v1.0.0").
	// Local paths have no meaningful tag, so we leave the spec version as-is.
	if !install.IsLocalPath(i.source) {
		i.spec.Version = registry.TagFromReference(i.source)
	}

	results, err := install.Install(i.layoutPath, i.source, i.spec, i.cfg)
	if err != nil {
		return ui.InstallResultMsg{Err: err}
	}
	if len(results) == 0 {
		return ui.InstallResultMsg{Err: fmt.Errorf("no tools were installed to")}
	}
	return ui.FormatInstallResults(results, i.spec)
}
