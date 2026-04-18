package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Push model phases.
const (
	pushPhaseResolving = iota
	pushPhasePushing
	pushPhaseDone
)

// PushResultMsg is sent when the async push operation completes.
type PushResultMsg struct {
	Digest   string // e.g. "sha256:abc123..."
	SpecType string // from manifest annotations (may be empty)
	Err      error
}

// PushResolveMsg is sent when the resolve phase completes.
type PushResolveMsg struct {
	Err error
}

// pushTickMsg signals that the minimum spinner duration for a phase has elapsed.
type pushTickMsg struct{}

// PushModel is the Bubble Tea model for the "taito push" command.
// It runs a two-phase pipeline: resolve → push → done.
type PushModel struct {
	spinner   spinner.Model
	phase     int
	phaseText string
	reference string

	// Async work functions injected by the caller.
	resolveFn func() tea.Msg
	pushFn    func() tea.Msg

	// Phase gating.
	tickDone       bool
	pendingResolve *PushResolveMsg
	pendingResult  *PushResultMsg

	// Terminal state.
	result *PushResultMsg
	done   bool
	err    error
	width  int // terminal width from WindowSizeMsg
}

// NewPushModel creates a PushModel.
//
// resolveFn validates the local OCI layout exists and authenticates.
// pushFn performs the actual oras.Copy to the remote registry.
func NewPushModel(reference string, resolveFn, pushFn func() tea.Msg) PushModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	return PushModel{
		spinner:   s,
		phase:     pushPhaseResolving,
		phaseText: "Resolving...",
		reference: reference,
		resolveFn: resolveFn,
		pushFn:    pushFn,
	}
}

// Err returns any error after the program exits.
func (m PushModel) Err() error {
	return m.err
}

func (m PushModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return pushTickMsg{} }),
		m.resolveFn,
	)
}

func (m PushModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case pushTickMsg:
		m.tickDone = true
		return m.tryAdvance()

	case PushResolveMsg:
		m.pendingResolve = &msg
		return m.tryAdvance()

	case PushResultMsg:
		m.pendingResult = &msg
		return m.tryAdvance()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m PushModel) tryAdvance() (tea.Model, tea.Cmd) {
	if !m.tickDone {
		return m, nil
	}

	// Phase 1: resolve completed.
	if m.phase == pushPhaseResolving && m.pendingResolve != nil {
		msg := m.pendingResolve
		m.pendingResolve = nil
		m.tickDone = false

		if msg.Err != nil {
			m.done = true
			m.err = msg.Err
			m.result = &PushResultMsg{Err: msg.Err}
			return m, tea.Quit
		}

		m.phase = pushPhasePushing
		m.phaseText = fmt.Sprintf("Pushing '%s'...", m.reference)

		return m, tea.Batch(
			tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return pushTickMsg{} }),
			m.pushFn,
		)
	}

	// Phase 2: push completed.
	if m.phase == pushPhasePushing && m.pendingResult != nil {
		msg := m.pendingResult
		m.pendingResult = nil
		m.tickDone = false

		m.done = true
		m.phase = pushPhaseDone
		m.result = msg
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

func (m PushModel) View() string {
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phaseText), m.width)
	}

	if m.result == nil {
		return ""
	}

	// Error.
	if m.result.Err != nil {
		return WrapView(fmt.Sprintf(" %s  Failed to push '%s'\n    %v\n",
			FailIcon(), m.reference, m.result.Err), m.width)
	}

	// Success.
	typeBadgeStr := ""
	if m.result.SpecType != "" {
		typeBadgeStr = TypeBadge(m.result.SpecType) + "  "
	}

	var b strings.Builder
	fmt.Fprintf(&b, " %s  %sPushed '%s'\n", SuccessIcon(), typeBadgeStr, m.reference)
	if m.result.Digest != "" {
		dimStyle := DimStyle()
		fmt.Fprintf(&b, "           %s\n", dimStyle.Render(m.result.Digest))
	}

	return WrapView(b.String(), m.width)
}
