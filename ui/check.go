package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CheckResultMsg is sent by the async check function when validation completes.
type CheckResultMsg struct {
	Err      error    // Hard validation or load error (nil = no hard error).
	Warnings []string // Non-fatal warnings from validation.
	SpecName string   // The name field from the loaded spec (empty on load error).
	SpecType string   // The type field from the loaded spec.
}

// CheckModel is the Bubble Tea model for the "taito check" command.
type CheckModel struct {
	spinner  spinner.Model
	specPath string
	result   *CheckResultMsg
	done     bool
	checkCmd func() tea.Msg
	width    int // terminal width from WindowSizeMsg
}

// NewCheckModel creates a new CheckModel.
func NewCheckModel(specPath string, checkCmd func() tea.Msg) CheckModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return CheckModel{
		spinner:  s,
		specPath: specPath,
		checkCmd: checkCmd,
	}
}

func (m CheckModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkCmd)
}

func (m CheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	case CheckResultMsg:
		m.done = true
		m.result = &msg
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m CheckModel) View() string {
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s Checking %s...\n\n", m.spinner.View(), m.specPath), m.width)
	}

	r := m.result

	typeBadgeStr := ""
	if r.SpecType != "" {
		typeBadgeStr = TypeBadge(r.SpecType) + "  "
	}

	warnStyle := lipgloss.NewStyle().Foreground(ColorWarning)

	// Hard error — could not load or validate at all.
	if r.Err != nil {
		return WrapView(fmt.Sprintf(" %s  Failed to validate\n    %v\n", FailIcon(), r.Err), m.width)
	}

	// Valid with warnings.
	if len(r.Warnings) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, " %s  %s'%s' is valid with warnings\n",
			SuccessIcon(), typeBadgeStr, r.SpecName)
		for _, w := range r.Warnings {
			b.WriteString(warnStyle.Render("           - "+w) + "\n")
		}
		return WrapView(b.String(), m.width)
	}

	// Clean pass.
	return WrapView(fmt.Sprintf(" %s  %s'%s' is valid\n", SuccessIcon(), typeBadgeStr, r.SpecName), m.width)
}
