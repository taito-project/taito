package ui

import (
	"fmt"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/spec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Install model phases.
const (
	installPhaseResolving = iota
	installPhaseExtracting
	installPhaseInstalling
	installPhaseDone
)

// InstallToolResult holds install paths for a single tool.
type InstallToolResult struct {
	Tool  string   // tool display name
	Paths []string // absolute install paths
}

// SelectableItem is an item shown in the interactive checklist.
type SelectableItem struct {
	ID       string // The underlying ID (e.g. include path or skill name)
	Name     string // Display name (e.g. "git-commit-helper")
	SpecType string // Display type (e.g. "skill" or "agent")
}

// InstallResultMsg is sent when the full install operation completes.
type InstallResultMsg struct {
	Name       string              // package name (from taito.spec)
	SpecType   string              // "skill", "agent", or "bundle"
	Tools      []InstallToolResult // per-tool install results
	SkillCount int
	AgentCount int
	Err        error
}

// InstallResolveMsg is sent when source resolution completes.
// LayoutPath is the path to the OCI layout to install from.
type InstallResolveMsg struct {
	LayoutPath string
	Warning    string                      // non-fatal warning from resolution (e.g. missing taito.spec)
	Items      []SelectableItem            // if populated, prompts the user to select skills
	OnSelect   func(chosen []string) error // callback to write fallback
	Err        error
}

// InstallExtractMsg is sent when spec extraction completes (we know the name
// and type). This is used to update the spinner text.
type InstallExtractMsg struct {
	Items    []SelectableItem
	OnSelect func(chosenIDs []string) error
	Warning  string

	Name     string
	SpecType string
	Err      error
}

// installTickMsg signals that the minimum spinner duration has elapsed.
type installTickMsg struct{}

// SkillsSelectedMsg is sent when the user finishes selecting skills.
type SkillsSelectedMsg struct {
	Chosen []string
}

// Installer defines the strategy for installing a package.
type Installer interface {
	Resolve() tea.Msg
	Extract() tea.Msg
	Install() tea.Msg
}

// --- Sub-component: SkillSelectorModel ---

type SkillSelectorModel struct {
	items          []SelectableItem
	selectedSkills map[int]bool
	cursorIndex    int
	source         string
	warning        string
	width          int
}

func NewSkillSelectorModel(source string, items []SelectableItem, warning string, width int) *SkillSelectorModel {
	selected := make(map[int]bool)
	for i := range items {
		selected[i] = true
	}
	return &SkillSelectorModel{
		items:          items,
		source:         source,
		selectedSkills: selected,
		warning:        warning,
		width:          width,
	}
}

func (m *SkillSelectorModel) Update(msg tea.Msg) (*SkillSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursorIndex > 0 {
				m.cursorIndex--
			}
		case "down", "j":
			if m.cursorIndex < len(m.items)-1 {
				m.cursorIndex++
			}
		case " ", "space":
			m.selectedSkills[m.cursorIndex] = !m.selectedSkills[m.cursorIndex]
		case "enter":
			var chosen []string
			for i, item := range m.items {
				if m.selectedSkills[i] {
					chosen = append(chosen, item.ID)
				}
			}
			return m, func() tea.Msg { return SkillsSelectedMsg{Chosen: chosen} }
		}
	}
	return m, nil
}

func (m *SkillSelectorModel) View() string {
	var b strings.Builder
	mainColor := lipgloss.NewStyle().Foreground(ColorPrimary)
	dimStyle := DimStyle()

	if m.warning != "" {
		fmt.Fprintf(&b, "\n %s  %s\n\n", WarningIcon(), m.warning)
	}

	fmt.Fprintf(&b, "Please select skills/agents to install from '%s':\n\n", m.source)

	for i, item := range m.items {
		cursor := " "
		if m.cursorIndex == i {
			cursor = mainColor.Render(">")
		}

		checked := " "
		displayName := item.Name
		displayType := item.SpecType

		if m.selectedSkills[i] {
			checked = mainColor.Render("x")
			displayName = mainColor.Render(displayName)
		} else {
			displayName = dimStyle.Render(displayName)
			displayType = dimStyle.Render(displayType)
		}

		fmt.Fprintf(&b, "%s [%s] %s - %s\n", cursor, checked, displayName, displayType)
	}

	b.WriteString("\n(Press Space to toggle, Enter to confirm)\n")
	return WrapView(b.String(), m.width)
}

// --- Main Component: InstallModel ---

// InstallModel is the Bubble Tea model for the "taito install" command.
// Phases: resolve → extract → install → done.
type InstallModel struct {
	spinner   spinner.Model
	phase     int
	phaseText string
	source    string // user-provided source (reference or local path)

	installer Installer

	// Phase gating.
	tickDone       bool
	pendingResolve *InstallResolveMsg
	pendingExtract *InstallExtractMsg

	// Skill selection state.
	skillSelector *SkillSelectorModel
	onSelect      func([]string) error

	// Terminal state.
	done   bool
	result *InstallResultMsg
	err    error
	width  int
}

// NewInstallModel creates an InstallModel.
func NewInstallModel(source string, installer Installer) InstallModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	s.Spinner = spinner.Dot

	return InstallModel{
		spinner:   s,
		phase:     installPhaseResolving,
		phaseText: "Resolving source...",
		source:    source,
		installer: installer,
	}
}

// Err returns any error after the program exits.
func (m InstallModel) Err() error {
	return m.err
}

func (m InstallModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return installTickMsg{} }),
		m.installer.Resolve,
	)
}

func (m InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If we get window size, update the width in models.
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		if m.skillSelector != nil {
			m.skillSelector.width = msg.Width
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}

		if m.skillSelector != nil {
			var cmd tea.Cmd
			m.skillSelector, cmd = m.skillSelector.Update(msg)
			return m, cmd
		}

	case SkillsSelectedMsg:
		if m.onSelect != nil {
			if err := m.onSelect(msg.Chosen); err != nil {
				m.err = err
				m.done = true
				m.result = &InstallResultMsg{Err: err}
				return m, tea.Quit
			}
		}

		m.skillSelector = nil
		m.onSelect = nil

		// Resume installation by simulating a resolved message
		m.pendingResolve = &InstallResolveMsg{}
		m.tickDone = true
		return m.tryAdvance()

	case installTickMsg:
		m.tickDone = true
		return m.tryAdvance()

	case InstallResolveMsg:
		m.pendingResolve = &msg
		return m.tryAdvance()

	case InstallExtractMsg:
		m.pendingExtract = &msg
		return m.tryAdvance()

	case InstallResultMsg:
		m.done = true
		m.phase = installPhaseDone
		m.result = &msg
		m.err = msg.Err
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m InstallModel) tryAdvance() (tea.Model, tea.Cmd) {
	if !m.tickDone {
		return m, nil
	}

	// Phase 1: resolve completed → start extracting.
	if m.phase == installPhaseResolving && m.pendingResolve != nil {
		msg := m.pendingResolve

		// Intercept: If there are available skills, pause and ask the user via the sub-component.
		if len(msg.Items) > 0 {
			m.skillSelector = NewSkillSelectorModel(m.source, msg.Items, msg.Warning, m.width)
			m.onSelect = msg.OnSelect

			// Clear the pending message so we stop advancing
			m.pendingResolve = nil
			m.tickDone = false
			return m, nil
		}

		m.pendingResolve = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.err = msg.Err
			m.result = &InstallResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		m.phase = installPhaseExtracting
		m.phaseText = "Extracting..."

		installer := m.installer
		return m, tea.Batch(
			tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return installTickMsg{} }),
			installer.Extract,
		)
	}

	// Phase 2: extract completed → start installing.
	if m.phase == installPhaseExtracting && m.pendingExtract != nil {
		msg := m.pendingExtract

		// Intercept if Extract provided items to select
		if len(msg.Items) > 0 {
			m.skillSelector = NewSkillSelectorModel(m.source, msg.Items, msg.Warning, m.width)
			m.onSelect = msg.OnSelect

			// Clear the items so we don't prompt again
			msg.Items = nil
			m.tickDone = false
			return m, nil
		}
		m.pendingExtract = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.err = msg.Err
			m.result = &InstallResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		// Update spinner to uniformly print "Installing '<source>'..."
		m.phaseText = fmt.Sprintf("Installing '%s'...", m.source)
		m.phase = installPhaseInstalling

		installer := m.installer
		return m, tea.Batch(
			tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return installTickMsg{} }),
			installer.Install,
		)
	}

	return m, nil
}

func (m InstallModel) View() string {
	if m.skillSelector != nil {
		return m.skillSelector.View()
	}

	// Active phases: resolving, extracting, installing.
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phaseText), m.width)
	}

	if m.result == nil {
		return ""
	}

	if m.result.Err != nil {
		return m.viewError()
	}

	return m.viewSuccess()
}

// viewError renders the install failure message.
func (m InstallModel) viewError() string {
	return WrapView(fmt.Sprintf(" %s  Failed to install '%s'\n    %v\n",
		FailIcon(), m.source, m.result.Err), m.width)
}

// viewSuccess renders the install success summary with per-tool paths.
func (m InstallModel) viewSuccess() string {
	var b strings.Builder
	dimStyle := DimStyle()

	typeBadge := TypeBadge(m.result.SpecType)
	toolCount := len(m.result.Tools)

	if m.result.SpecType == "bundle" {
		summary := bundleSummary(m.result.SkillCount, m.result.AgentCount)
		fmt.Fprintf(&b, " %s  %s  '%s' (%s) installed to %d tool%s\n",
			SuccessIcon(), typeBadge, m.result.Name, summary,
			toolCount, plural(toolCount))
	} else {
		fmt.Fprintf(&b, " %s  %s  '%s' installed to %d tool%s\n",
			SuccessIcon(), typeBadge, m.result.Name,
			toolCount, plural(toolCount))
	}

	writeToolPaths(&b, m.result.Tools, dimStyle)

	return WrapView(b.String(), m.width)
}

// bundleSummary returns a human-readable summary like "2 skills, 1 agent".
func bundleSummary(skillCount, agentCount int) string {
	var parts []string
	if skillCount > 0 {
		parts = append(parts, fmt.Sprintf("%d skill%s", skillCount, plural(skillCount)))
	}
	if agentCount > 0 {
		parts = append(parts, fmt.Sprintf("%d agent%s", agentCount, plural(agentCount)))
	}
	return strings.Join(parts, ", ")
}

// writeToolPaths writes the per-tool path listing to a string builder.
func writeToolPaths(b *strings.Builder, tools []InstallToolResult, dimStyle lipgloss.Style) {
	for _, tr := range tools {
		for i, p := range tr.Paths {
			if i == 0 {
				label := fmt.Sprintf("    %-14s", tr.Tool)
				b.WriteString(dimStyle.Render(label) + dimStyle.Render(p) + "\n")
			} else {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %-14s%s", "", p)) + "\n")
			}
		}
	}
}

// plural returns "s" if n != 1.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// FormatInstallResults converts backend install results into the UI message type.
func FormatInstallResults(results []install.InstallResult, s *spec.TaitoSpec) InstallResultMsg {
	var name, specType string
	if s != nil {
		name = s.Name
		specType = s.Type
	}

	toolMap := make(map[string]*InstallToolResult)
	var toolOrder []string
	skillCount := 0
	agentCount := 0

	for _, r := range results {
		tr, ok := toolMap[r.Tool]
		if !ok {
			tr = &InstallToolResult{Tool: r.Tool}
			toolMap[r.Tool] = tr
			toolOrder = append(toolOrder, r.Tool)
		}
		tr.Paths = append(tr.Paths, r.Path)

		switch r.SpecType {
		case spec.TypeSkill:
			skillCount++
		case spec.TypeAgent:
			agentCount++
		}
	}

	if specType != spec.TypeBundle {
		skillCount = 0
		agentCount = 0
	} else {
		toolCount := len(toolOrder)
		if toolCount > 0 {
			skillCount = skillCount / toolCount
			agentCount = agentCount / toolCount
		}
	}

	var tools []InstallToolResult
	for _, tname := range toolOrder {
		tools = append(tools, *toolMap[tname])
	}

	return InstallResultMsg{
		Name:       name,
		SpecType:   specType,
		Tools:      tools,
		SkillCount: skillCount,
		AgentCount: agentCount,
	}
}
