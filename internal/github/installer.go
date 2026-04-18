package github

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/spec"
	"github.com/taito-project/taito/ui"
)

// Installer implements ui.Installer for GitHub sources.
type Installer struct {
	source    string
	cfg       *config.Config
	sourceDir string
	workspace string
	spec      *spec.TaitoSpec
	warning   string
	version   string // resolved version: user-specified tag/ref, or commit SHA if none given
}

func NewInstaller(source string, cfg *config.Config) *Installer {
	return &Installer{
		source: source,
		cfg:    cfg,
	}
}

func (i *Installer) Resolve() tea.Msg {
	result, err := Resolve(i.source)
	if err != nil {
		return ui.InstallResolveMsg{Err: err}
	}

	i.sourceDir = result.SourceDir
	i.workspace = result.Workspace
	i.warning = result.Warning
	i.version = result.Version

	return ui.InstallResolveMsg{}
}

func (i *Installer) Extract() tea.Msg {
	s, err := ReadSpec(i.sourceDir)
	if err != nil {
		return ui.InstallExtractMsg{Err: err}
	}
	i.spec = s

	var items []ui.SelectableItem
	if s.Type == spec.TypeBundle {
		for _, inc := range s.Includes {
			clean := filepath.Clean(inc)
			childSpecPath := filepath.Join(i.sourceDir, clean)
			childDir := filepath.Dir(childSpecPath)

			childSpec, err := ReadSpec(childDir)
			if err == nil {
				items = append(items, ui.SelectableItem{
					ID:       inc,
					Name:     childSpec.Name,
					SpecType: childSpec.Type,
				})
			}
		}
	}

	return ui.InstallExtractMsg{
		Name:     s.Name,
		SpecType: s.Type,
		Items:    items,
		Warning:  i.warning,
		OnSelect: func(chosenIDs []string) error {
			i.spec.Includes = chosenIDs
			return nil
		},
	}
}

func (i *Installer) Install() tea.Msg {
	// Override the spec version with the resolved version so that
	// installed.json always reflects what was actually installed.
	// This will be the user-specified tag/ref (e.g. "v2.0.0") when
	// provided, or the commit SHA when no tag was given.
	if i.version != "" {
		i.spec.Version = i.version
	}

	results, err := Install(i.sourceDir, i.source, i.spec, i.cfg)
	if err != nil {
		return ui.InstallResultMsg{Err: err}
	}
	if len(results) == 0 {
		return ui.InstallResultMsg{Err: fmt.Errorf("no tools were installed to")}
	}
	return ui.FormatInstallResults(results, i.spec)
}

func (i *Installer) Cleanup() {
	if i.workspace != "" {
		_ = os.RemoveAll(i.workspace)
	}
}
