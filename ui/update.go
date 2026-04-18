package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Update model phases.
const (
	updatePhaseChecking = iota
	updatePhaseResults
	updatePhaseUpdating
	updatePhaseDone
)

// UpdateEntry holds the data the UI needs for displaying a single package's
// update status. This is populated by the caller (cmd/update.go) to avoid
// import cycles with internal/update.
type UpdateEntry struct {
	ID             string
	Name           string
	SpecType       string
	CurrentVersion string
	LatestVersion  string
	Reference      string
	UpdateRef      string // reference with the new version for reinstall
	HasUpdate      bool
	IsLocal        bool
	Error          error
	IsBundleChild  bool
	BundleID       string
}

// UpdateCheckMsg is sent when the version check completes.
type UpdateCheckMsg struct {
	Entries []UpdateEntry
	Err     error
}

// UpdateInstallMsg is sent when a single package update completes.
type UpdateInstallMsg struct {
	ID   string
	Name string
	Err  error
}

// updateTickMsg signals the minimum spinner duration has elapsed.
type updateTickMsg struct{}

// UpdateModel is the Bubble Tea model for the "taito update" command.
type UpdateModel struct {
	spinner   spinner.Model
	phase     int
	phaseText string

	// Check phase.
	checkFn      func() tea.Msg
	tickDone     bool
	pendingCheck *UpdateCheckMsg

	// Results.
	entries   []UpdateEntry
	updatable []UpdateEntry
	filterID  string

	// Update phase.
	installFn     func(ref string) tea.Msg
	updateQueue   []UpdateEntry
	updateCurrent int
	updateResults []UpdateInstallMsg

	// Terminal state.
	done  bool
	err   error
	width int
}

// NewUpdateModel creates an UpdateModel.
func NewUpdateModel(
	filterID string,
	checkFn func() tea.Msg,
	installFn func(ref string) tea.Msg,
) UpdateModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	phaseText := "Checking for updates..."
	if filterID != "" {
		phaseText = fmt.Sprintf("Checking for updates (%s)...", filterID)
	}

	return UpdateModel{
		spinner:   s,
		phase:     updatePhaseChecking,
		phaseText: phaseText,
		checkFn:   checkFn,
		installFn: installFn,
		filterID:  filterID,
	}
}

// Err returns any error after the program exits.
func (m UpdateModel) Err() error {
	return m.err
}

func (m UpdateModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return updateTickMsg{} }),
		m.checkFn,
	)
}

func (m UpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}

		// Handle confirmation prompt: y → proceed, n/Enter → cancel.
		if m.phase == updatePhaseResults && len(m.updatable) > 0 {
			switch msg.String() {
			case "y", "Y":
				return m.startUpdating()
			case "n", "N", "enter":
				m.done = true
				return m, tea.Quit
			}
		}

	case updateTickMsg:
		m.tickDone = true
		return m.tryAdvanceCheck()

	case UpdateCheckMsg:
		m.pendingCheck = &msg
		return m.tryAdvanceCheck()

	case UpdateInstallMsg:
		m.updateResults = append(m.updateResults, msg)
		m.updateCurrent++

		if m.updateCurrent < len(m.updateQueue) {
			// Start the next update.
			next := m.updateQueue[m.updateCurrent]
			m.phaseText = fmt.Sprintf("Updating '%s' to %s...", next.Name, next.LatestVersion)
			installFn := m.installFn
			ref := next.UpdateRef
			return m, func() tea.Msg { return installFn(ref) }
		}

		// All done.
		m.phase = updatePhaseDone
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m UpdateModel) tryAdvanceCheck() (tea.Model, tea.Cmd) {
	if !m.tickDone || m.pendingCheck == nil {
		return m, nil
	}

	msg := m.pendingCheck
	m.pendingCheck = nil
	m.tickDone = false

	if msg.Err != nil {
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	m.entries = msg.Entries
	for _, e := range msg.Entries {
		if e.HasUpdate && !e.IsBundleChild && e.Error == nil {
			m.updatable = append(m.updatable, e)
		}
	}
	m.phase = updatePhaseResults

	// If nothing to update, quit immediately.
	if len(m.updatable) == 0 {
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m UpdateModel) startUpdating() (tea.Model, tea.Cmd) {
	m.phase = updatePhaseUpdating
	m.updateQueue = m.updatable
	m.updateCurrent = 0

	first := m.updateQueue[0]
	m.phaseText = fmt.Sprintf("Updating '%s' to %s...", first.Name, first.LatestVersion)

	installFn := m.installFn
	ref := first.UpdateRef
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg { return installFn(ref) },
	)
}

func (m UpdateModel) View() string {
	// Checking phase — spinner.
	if m.phase == updatePhaseChecking {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phaseText), m.width)
	}

	// Updating phase — spinner.
	if m.phase == updatePhaseUpdating {
		var b strings.Builder
		fmt.Fprintf(&b, "\n %s %s\n", m.spinner.View(), m.phaseText)

		// Show completed updates.
		for _, ur := range m.updateResults {
			if ur.Err != nil {
				fmt.Fprintf(&b, " %s  Failed to update '%s': %v\n", FailIcon(), ur.Name, ur.Err)
			} else {
				fmt.Fprintf(&b, " %s  Updated '%s'\n", SuccessIcon(), ur.Name)
			}
		}
		b.WriteString("\n")
		return WrapView(b.String(), m.width)
	}

	// Results phase — show table + confirmation.
	if m.phase == updatePhaseResults {
		return m.viewResults()
	}

	// Done phase.
	if m.done {
		return m.viewDone()
	}

	return ""
}

// isLastBundleChild reports whether target is the last bundle child in its group.
func isLastBundleChild(entries []UpdateEntry, target UpdateEntry) bool {
	foundSelf := false
	for _, other := range entries {
		if other.BundleID != target.BundleID || !other.IsBundleChild {
			continue
		}
		if other.ID == target.ID {
			foundSelf = true
			continue
		}
		if foundSelf {
			return false
		}
	}
	return true
}

// formatVersionColumn renders the version cell for a single entry row.
func formatVersionColumn(r UpdateEntry) string {
	dimStyle := DimStyle()
	updateStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	upToDateStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
	errorStyle := lipgloss.NewStyle().Foreground(ColorError)

	switch {
	case r.Error != nil:
		return r.CurrentVersion + "  " + errorStyle.Render(fmt.Sprintf("error: %v", r.Error))
	case r.IsLocal:
		return r.CurrentVersion + "  " + dimStyle.Render("local")
	case r.HasUpdate:
		return dimStyle.Render(r.CurrentVersion) + " " + dimStyle.Render("→") + " " + updateStyle.Render(r.LatestVersion)
	default:
		return r.CurrentVersion + "  " + upToDateStyle.Render("up to date")
	}
}

// buildEntryRows converts update entries into table row data.
func buildEntryRows(entries []UpdateEntry) [][]string {
	var rows [][]string
	for _, r := range entries {
		if r.IsBundleChild {
			prefix := " ├─ "
			if isLastBundleChild(entries, r) {
				prefix = " ╰─ "
			}
			rows = append(rows, []string{"", r.SpecType, prefix + r.Name, ""})
			continue
		}

		displayID := r.ID
		if len(displayID) > 8 {
			displayID = displayID[:8]
		}

		rows = append(rows, []string{displayID, r.SpecType, r.Name, formatVersionColumn(r)})
	}
	return rows
}

func (m UpdateModel) viewResults() string {
	var b strings.Builder
	b.WriteString("\n")

	headerStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		BorderRow(false).
		BorderColumn(false).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderHeader(false).
		Headers("ID", "TYPE", "NAME", "VERSION").
		Rows(buildEntryRows(m.entries)...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	b.WriteString(t.Render())
	b.WriteString("\n\n")

	updateCount := len(m.updatable)
	if updateCount == 0 {
		fmt.Fprintf(&b, " %s  All packages are up to date\n", SuccessIcon())
		return WrapView(b.String(), m.width)
	}

	// Confirmation prompt.
	fmt.Fprintf(&b, " %d update%s available. Proceed? [y/N] ", updateCount, plural(updateCount))
	b.WriteString("\n")

	// Hint for single-package update when checking all packages.
	if m.filterID == "" && updateCount > 0 && len(m.entries) > 1 {
		b.WriteString(DimStyle().Render(" Tip: to update a single package, use 'taito update <id>'"))
		b.WriteString("\n")
	}

	return WrapView(b.String(), m.width)
}

// formatUpdateResultLine renders a single update result line.
func formatUpdateResultLine(ur UpdateInstallMsg, orig *UpdateEntry) string {
	if ur.Err != nil {
		return fmt.Sprintf(" %s  Failed to update '%s'\n    %v\n", FailIcon(), ur.Name, ur.Err)
	}
	if orig != nil {
		typeBadge := TypeBadge(orig.SpecType)
		return fmt.Sprintf(" %s  %s  '%s' updated %s → %s\n",
			SuccessIcon(), typeBadge, ur.Name,
			orig.CurrentVersion, orig.LatestVersion)
	}
	return fmt.Sprintf(" %s  Updated '%s'\n", SuccessIcon(), ur.Name)
}

// renderUpdateSummary renders the post-update summary with per-package results.
func renderUpdateSummary(results []UpdateInstallMsg, queue []UpdateEntry) string {
	var b strings.Builder
	b.WriteString("\n")

	var successCount, failCount int
	for i, ur := range results {
		var orig *UpdateEntry
		if i < len(queue) {
			orig = &queue[i]
		}
		b.WriteString(formatUpdateResultLine(ur, orig))

		if ur.Err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	if failCount > 0 && successCount > 0 {
		fmt.Fprintf(&b, "\n %d updated, %d failed\n", successCount, failCount)
	}
	return b.String()
}

func (m UpdateModel) viewDone() string {
	if len(m.entries) == 0 {
		return WrapView(fmt.Sprintf(" %s  No packages installed\n", SuccessIcon()), m.width)
	}

	if len(m.updatable) == 0 && len(m.updateResults) == 0 {
		return m.viewResults()
	}

	if len(m.updateResults) > 0 {
		return WrapView(renderUpdateSummary(m.updateResults, m.updateQueue), m.width)
	}

	return ""
}
