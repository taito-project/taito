package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pull model phases.
const (
	pullPhaseResolving = iota
	pullPhasePulling
	pullPhaseValidating
	pullPhaseDone
)

// PullResultMsg is sent when the async pull + validate operation completes.
type PullResultMsg struct {
	Digest   string   // e.g. "sha256:abc123..."
	SpecType string   // from validated dev.taito.spec.type annotation
	Warnings []string // non-fatal warnings from validation
	Err      error
}

// PullResolveMsg is sent when the resolve phase completes.
type PullResolveMsg struct {
	Err error
}

// PullCopyMsg is sent when the copy (download) phase completes.
type PullCopyMsg struct {
	Digest   string
	SpecType string
	Warnings []string
	Err      error
}

// pullTickMsg signals that the minimum spinner duration for a phase has elapsed.
type pullTickMsg struct{}

// PullModel is the Bubble Tea model for the "taito pull" command.
// It runs a three-phase pipeline: resolve → pull → validate → done.
// The pull and validate phases are combined in the async work because
// validation requires the pulled data; the validate spinner is a visual
// representation of the final validation step.
type PullModel struct {
	spinner   spinner.Model
	phase     int
	phaseText string
	reference string

	// Async work functions injected by the caller.
	resolveFn func() tea.Msg
	pullFn    func() tea.Msg // performs pull + validation atomically

	// Phase gating.
	tickDone       bool
	pendingResolve *PullResolveMsg
	pendingCopy    *PullCopyMsg

	// Terminal state.
	result *PullResultMsg
	done   bool
	err    error
	width  int // terminal width from WindowSizeMsg
}

// NewPullModel creates a PullModel.
//
// resolveFn authenticates and resolves the reference on the remote.
// pullFn performs oras.Copy + ValidateTaitoArtifact (discard on failure).
func NewPullModel(reference string, resolveFn, pullFn func() tea.Msg) PullModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return PullModel{
		spinner:   s,
		phase:     pullPhaseResolving,
		phaseText: "Resolving...",
		reference: reference,
		resolveFn: resolveFn,
		pullFn:    pullFn,
	}
}

// Err returns any error after the program exits.
func (m PullModel) Err() error {
	return m.err
}

// SpecType returns the validated spec type from the pulled artifact.
func (m PullModel) SpecType() string {
	if m.result != nil {
		return m.result.SpecType
	}
	return ""
}

func (m PullModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return pullTickMsg{} }),
		m.resolveFn,
	)
}

func (m PullModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case pullTickMsg:
		m.tickDone = true
		return m.tryAdvance()

	case PullResolveMsg:
		m.pendingResolve = &msg
		return m.tryAdvance()

	case PullCopyMsg:
		m.pendingCopy = &msg
		return m.tryAdvance()

	case PullResultMsg:
		// Final phase transition: validating → done.
		m.done = true
		m.phase = pullPhaseDone
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

func (m PullModel) tryAdvance() (tea.Model, tea.Cmd) {
	if !m.tickDone {
		return m, nil
	}

	// Phase 1: resolve completed.
	if m.phase == pullPhaseResolving && m.pendingResolve != nil {
		msg := m.pendingResolve
		m.pendingResolve = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.err = msg.Err
			m.result = &PullResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		m.phase = pullPhasePulling
		m.phaseText = fmt.Sprintf("Pulling '%s'...", m.reference)

		return m, tea.Batch(
			tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return pullTickMsg{} }),
			m.pullFn,
		)
	}

	// Phase 2: pull + validate completed.
	if m.phase == pullPhasePulling && m.pendingCopy != nil {
		msg := m.pendingCopy
		m.pendingCopy = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.phase = pullPhaseDone
			m.err = msg.Err
			m.result = &PullResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		// Show validating phase briefly.
		m.phase = pullPhaseValidating
		m.phaseText = "Validating artifact..."

		// Validation already happened inside pullFn, so we just show
		// the spinner for the minimum duration and then finish.
		stashedResult := &PullResultMsg{
			Digest:   msg.Digest,
			SpecType: msg.SpecType,
			Warnings: msg.Warnings,
		}

		return m, tea.Tick(minPhaseDuration, func(time.Time) tea.Msg {
			return *stashedResult
		})
	}

	// Phase 3: validating complete — handled by PullResultMsg in Update().

	return m, nil
}

func (m PullModel) View() string {
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phaseText), m.width)
	}

	if m.result == nil {
		return ""
	}

	// Error.
	if m.result.Err != nil {
		return WrapView(fmt.Sprintf(" %s  Failed to pull '%s'\n    %v\n",
			FailIcon(), m.reference, m.result.Err), m.width)
	}

	// Success.
	typeBadgeStr := ""
	if m.result.SpecType != "" {
		typeBadgeStr = TypeBadge(m.result.SpecType) + "  "
	}

	warnStyle := lipgloss.NewStyle().Foreground(ColorWarning)
	dimStyle := DimStyle()

	var b strings.Builder

	if len(m.result.Warnings) > 0 {
		fmt.Fprintf(&b, " %s  %sPulled '%s' with warnings\n",
			SuccessIcon(), typeBadgeStr, m.reference)
		if m.result.Digest != "" {
			fmt.Fprintf(&b, "           %s\n", dimStyle.Render(m.result.Digest))
		}
		for _, w := range m.result.Warnings {
			b.WriteString(warnStyle.Render("           - "+w) + "\n")
		}
	} else {
		fmt.Fprintf(&b, " %s  %sPulled '%s'\n", SuccessIcon(), typeBadgeStr, m.reference)
		if m.result.Digest != "" {
			fmt.Fprintf(&b, "           %s\n", dimStyle.Render(m.result.Digest))
		}
	}

	return WrapView(b.String(), m.width)
}
