package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockInstaller is a test installer that returns configurable messages.
type mockInstaller struct {
	resolveMsg tea.Msg
	extractMsg tea.Msg
	installMsg tea.Msg
}

func (m *mockInstaller) Resolve() tea.Msg { return m.resolveMsg }
func (m *mockInstaller) Extract() tea.Msg { return m.extractMsg }
func (m *mockInstaller) Install() tea.Msg { return m.installMsg }

func TestNonInteractive_SkipsSelectionOnResolve(t *testing.T) {
	var selectedIDs []string

	items := []SelectableItem{
		{ID: "skills/a/taito.spec", Name: "skill-a", SpecType: "skill"},
		{ID: "agents/b/taito.spec", Name: "agent-b", SpecType: "agent"},
	}

	installer := &mockInstaller{
		resolveMsg: InstallResolveMsg{
			Items: items,
			OnSelect: func(chosen []string) error {
				selectedIDs = chosen
				return nil
			},
		},
		extractMsg: InstallExtractMsg{Name: "test-bundle", SpecType: "bundle"},
		installMsg: InstallResultMsg{Name: "test-bundle", SpecType: "bundle"},
	}

	m := NewInstallModel("test-source", installer, true)

	// Simulate the tick completing and resolve message arriving.
	m.tickDone = true
	m.pendingResolve = &InstallResolveMsg{
		Items: items,
		OnSelect: func(chosen []string) error {
			selectedIDs = chosen
			return nil
		},
	}

	model, _ := m.tryAdvance()
	im := model.(InstallModel)

	// Should NOT show the skill selector.
	if im.skillSelector != nil {
		t.Fatal("expected skillSelector to be nil in non-interactive mode")
	}

	// Should have selected all items.
	if len(selectedIDs) != 2 {
		t.Fatalf("expected 2 selected IDs, got %d", len(selectedIDs))
	}
	if selectedIDs[0] != "skills/a/taito.spec" || selectedIDs[1] != "agents/b/taito.spec" {
		t.Errorf("unexpected selected IDs: %v", selectedIDs)
	}
}

func TestNonInteractive_SkipsSelectionOnExtract(t *testing.T) {
	var selectedIDs []string

	items := []SelectableItem{
		{ID: "skills/x/taito.spec", Name: "skill-x", SpecType: "skill"},
	}

	installer := &mockInstaller{
		extractMsg: InstallExtractMsg{
			Name:     "test-bundle",
			SpecType: "bundle",
			Items:    items,
			OnSelect: func(chosen []string) error {
				selectedIDs = chosen
				return nil
			},
		},
		installMsg: InstallResultMsg{Name: "test-bundle", SpecType: "bundle"},
	}

	m := NewInstallModel("test-source", installer, true)
	m.phase = installPhaseExtracting
	m.tickDone = true
	m.pendingExtract = &InstallExtractMsg{
		Name:     "test-bundle",
		SpecType: "bundle",
		Items:    items,
		OnSelect: func(chosen []string) error {
			selectedIDs = chosen
			return nil
		},
	}

	model, _ := m.tryAdvance()
	im := model.(InstallModel)

	if im.skillSelector != nil {
		t.Fatal("expected skillSelector to be nil in non-interactive mode")
	}

	if len(selectedIDs) != 1 || selectedIDs[0] != "skills/x/taito.spec" {
		t.Errorf("unexpected selected IDs: %v", selectedIDs)
	}
}

func TestInteractive_ShowsSelector(t *testing.T) {
	items := []SelectableItem{
		{ID: "skills/a/taito.spec", Name: "skill-a", SpecType: "skill"},
	}

	installer := &mockInstaller{}

	m := NewInstallModel("test-source", installer, false)
	m.tickDone = true
	m.pendingResolve = &InstallResolveMsg{
		Items:    items,
		OnSelect: func(chosen []string) error { return nil },
	}

	model, _ := m.tryAdvance()
	im := model.(InstallModel)

	if im.skillSelector == nil {
		t.Fatal("expected skillSelector to be set in interactive mode")
	}
}
