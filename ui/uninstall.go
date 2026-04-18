package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Uninstall model phases.
const (
	uninstallPhaseLookup = iota
	uninstallPhaseRemoving
	uninstallPhaseDone
)

// UninstallToolResult groups removed paths for a single tool.
type UninstallToolResult struct {
	Tool  string   // tool display name
	Paths []string // absolute paths that were removed
}

// UninstallResultMsg is sent when the full uninstall operation completes.
type UninstallResultMsg struct {
	Name       string                // package name requested
	SpecType   string                // type of the removed package (or first child)
	Tools      []UninstallToolResult // per-tool removal results
	TotalCount int                   // total entries removed
	Err        error
}

// UninstallLookupMsg is sent when the lookup phase completes (found entries
// to remove, or not found).
type UninstallLookupMsg struct {
	Name  string // resolved package name
	Count int    // number of entries found
	Err   error
}

// uninstallTickMsg signals that the minimum spinner duration has elapsed.
type uninstallTickMsg struct{}

// UninstallModel is the Bubble Tea model for the "taito uninstall" command.
// Phases: lookup → remove → done.
type UninstallModel struct {
	spinner   spinner.Model
	phase     int
	phaseText string
	name      string // user-provided package name

	// Async work functions injected by the caller.
	lookupFn    func() tea.Msg
	uninstallFn func() tea.Msg

	// Phase gating.
	tickDone      bool
	pendingLookup *UninstallLookupMsg

	// Terminal state.
	result *UninstallResultMsg
	done   bool
	err    error
	width  int
}

// NewUninstallModel creates an UninstallModel.
//
// lookupFn checks installed.json for matching entries.
// uninstallFn performs the actual removal (directories + index update).
func NewUninstallModel(
	name string,
	lookupFn func() tea.Msg,
	uninstallFn func() tea.Msg,
) UninstallModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return UninstallModel{
		spinner:     s,
		phase:       uninstallPhaseLookup,
		phaseText:   "Looking up package...",
		name:        name,
		lookupFn:    lookupFn,
		uninstallFn: uninstallFn,
	}
}

// Err returns any error after the program exits.
func (m UninstallModel) Err() error {
	return m.err
}

func (m UninstallModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return uninstallTickMsg{} }),
		m.lookupFn,
	)
}

func (m UninstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case uninstallTickMsg:
		m.tickDone = true
		return m.tryAdvance()

	case UninstallLookupMsg:
		m.pendingLookup = &msg
		return m.tryAdvance()

	case UninstallResultMsg:
		m.done = true
		m.phase = uninstallPhaseDone
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

func (m UninstallModel) tryAdvance() (tea.Model, tea.Cmd) {
	if !m.tickDone {
		return m, nil
	}

	// Phase 1: lookup completed → start removing.
	if m.phase == uninstallPhaseLookup && m.pendingLookup != nil {
		msg := m.pendingLookup
		m.pendingLookup = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.err = msg.Err
			m.result = &UninstallResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		m.phase = uninstallPhaseRemoving
		m.phaseText = fmt.Sprintf("Uninstalling '%s'...", m.name)

		uninstallFn := m.uninstallFn
		return m, tea.Batch(
			tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return uninstallTickMsg{} }),
			uninstallFn,
		)
	}

	return m, nil
}

func (m UninstallModel) View() string {
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phaseText), m.width)
	}

	if m.result == nil {
		return ""
	}

	// Error.
	if m.result.Err != nil {
		return WrapView(fmt.Sprintf(" %s  Failed to uninstall '%s'\n    %v\n",
			FailIcon(), m.name, m.result.Err), m.width)
	}

	// Success.
	dimStyle := DimStyle()
	var b strings.Builder

	typeBadge := TypeBadge(m.result.SpecType)
	toolCount := len(m.result.Tools)

	fmt.Fprintf(&b, " %s  %s  Uninstalled '%s' from %d tool%s\n",
		SuccessIcon(), typeBadge, m.name,
		toolCount, plural(toolCount))

	// Per-tool path listing.
	for _, tr := range m.result.Tools {
		for i, p := range tr.Paths {
			if i == 0 {
				label := fmt.Sprintf("    %-14s", tr.Tool)
				b.WriteString(dimStyle.Render(label) + dimStyle.Render(p) + "\n")
			} else {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %-14s%s", "", p)) + "\n")
			}
		}
	}

	return WrapView(b.String(), m.width)
}
