package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/spec"
)

// minPhaseDuration is the minimum time each spinner phase is displayed.
const minPhaseDuration = 300 * time.Millisecond

// --- Internal message types for the multi-phase pipeline ---

type specLoadedMsg struct {
	Spec *spec.TaitoSpec
	Err  error
}

type specValidatedMsg struct {
	Warnings []spec.Warning
	Err      error
}

// phaseTickMsg signals that the minimum display time for a phase has elapsed.
type phaseTickMsg struct{}

// PackageResultMsg is sent when the archive creation step finishes.
type PackageResultMsg struct {
	Err error
}

// PackageModel is the Bubble Tea model for the "taito package" command.
// It runs a three-phase pipeline: load spec -> validate -> package.
//
// Each phase is displayed for at least minPhaseDuration. The model fires
// both the async work command and a tea.Tick in parallel, and only
// transitions to the next phase once both have completed.
type PackageModel struct {
	spinner    spinner.Model
	phase      string // current spinner text
	specPath   string // path to the taito.spec file
	contextDir string // source directory to package
	format     string // "oci" or "tar.gz"
	userRef    string // positional OCI reference the user provided (may be empty)

	// resolveTarget maps a reference and format to a full output path.
	// Injected by the caller so that the model does not need access to config.
	resolveTarget func(reference, format string) (string, error)

	// Populated during the pipeline.
	spec       *spec.TaitoSpec
	reference  string   // derived or user-provided OCI reference
	targetPath string   // resolved output path
	warnings   []string // validation warnings (formatted)

	// Phase gating: both the work result and tick must arrive before
	// transitioning. The stashed result is held until the tick fires
	// (or vice versa).
	tickDone         bool
	pendingLoaded    *specLoadedMsg
	pendingValidated *specValidatedMsg
	pendingResult    *PackageResultMsg

	// Terminal state.
	err   error
	done  bool
	width int // terminal width from WindowSizeMsg
}

// NewPackageModel creates a PackageModel.
//
// resolveTarget is a closure that turns a reference and format into an absolute
// path where the output should be written. It is called during the
// validate→package transition.
func NewPackageModel(
	specPath, contextDir, format, userRef string,
	resolveTarget func(reference, format string) (string, error),
) PackageModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	return PackageModel{
		spinner:       s,
		phase:         "Loading spec...",
		specPath:      specPath,
		contextDir:    contextDir,
		format:        format,
		userRef:       userRef,
		resolveTarget: resolveTarget,
	}
}

// Err returns the final error (if any) after the program exits.
func (m PackageModel) Err() error {
	return m.err
}

// Reference returns the resolved OCI reference after packaging completes.
func (m PackageModel) Reference() string {
	return m.reference
}

// SpecType returns the spec type (skill/agent/bundle) if available.
func (m PackageModel) SpecType() string {
	if m.spec != nil {
		return m.spec.Type
	}
	return ""
}

func (m PackageModel) Init() tea.Cmd {
	specPath := m.specPath
	return tea.Batch(
		m.spinner.Tick,
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return phaseTickMsg{} }),
		func() tea.Msg {
			s, err := spec.Load(specPath)
			return specLoadedMsg{Spec: s, Err: err}
		},
	)
}

func (m PackageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}

	case phaseTickMsg:
		m.tickDone = true
		return m.tryAdvance()

	case specLoadedMsg:
		m.pendingLoaded = &msg
		return m.tryAdvance()

	case specValidatedMsg:
		m.pendingValidated = &msg
		return m.tryAdvance()

	case PackageResultMsg:
		m.pendingResult = &msg
		return m.tryAdvance()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// tryAdvance checks whether both the phase tick and the current phase's work
// result have arrived. If so, it transitions to the next phase (or finishes).
func (m PackageModel) tryAdvance() (tea.Model, tea.Cmd) {
	if !m.tickDone {
		return m, nil
	}
	if m.pendingLoaded != nil {
		return m.advanceFromLoaded()
	}
	if m.pendingValidated != nil {
		return m.advanceFromValidated()
	}
	if m.pendingResult != nil {
		return m.advanceFromResult()
	}
	return m, nil
}

// advanceFromLoaded transitions from the spec-loaded phase to validation.
func (m PackageModel) advanceFromLoaded() (tea.Model, tea.Cmd) {
	msg := m.pendingLoaded
	m.pendingLoaded = nil
	m.tickDone = false

	if msg.Err != nil {
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	m.spec = msg.Spec
	m.phase = "Validating spec..."

	loaded := msg.Spec
	return m, tea.Batch(
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return phaseTickMsg{} }),
		func() tea.Msg {
			warnings, err := spec.Validate(loaded)
			return specValidatedMsg{Warnings: warnings, Err: err}
		},
	)
}

// deriveReference returns the OCI reference for the package, using the
// user-provided ref or falling back to name:version.
func (m PackageModel) deriveReference() string {
	if m.userRef != "" {
		return m.userRef
	}
	tag := "latest"
	if m.spec.Version != "" {
		tag = m.spec.Version
	}
	return fmt.Sprintf("%s:%s", m.spec.Name, tag)
}

// advanceFromValidated transitions from the validated phase to packaging.
func (m PackageModel) advanceFromValidated() (tea.Model, tea.Cmd) {
	msg := m.pendingValidated
	m.pendingValidated = nil
	m.tickDone = false

	if msg.Err != nil {
		m.done = true
		m.err = msg.Err
		return m, tea.Quit
	}

	// Store formatted warnings.
	for _, w := range msg.Warnings {
		m.warnings = append(m.warnings, fmt.Sprintf("%s: %s", w.Field, w.Message))
	}

	m.reference = m.deriveReference()

	// Resolve target path using the full reference and format.
	targetPath, err := m.resolveTarget(m.reference, m.format)
	if err != nil {
		m.done = true
		m.err = err
		return m, tea.Quit
	}
	m.targetPath = targetPath

	// Update spinner phase.
	formatLabel := "OCI layout"
	if m.format == "tar.gz" {
		formatLabel = "tar.gz archive"
	}
	m.phase = fmt.Sprintf("Packaging '%s' as %s...", m.reference, formatLabel)

	// Fire the archive creation.
	contextDir := m.contextDir
	target := m.targetPath
	ref := m.reference
	format := m.format
	s := m.spec
	return m, tea.Batch(
		tea.Tick(minPhaseDuration, func(time.Time) tea.Msg { return phaseTickMsg{} }),
		func() tea.Msg {
			var err error
			switch format {
			case "oci":
				err = archive.CreateOCILayout(contextDir, target, ref, s)
			default:
				err = archive.CreateTarGz(contextDir, target, s)
			}
			return PackageResultMsg{Err: err}
		},
	)
}

// advanceFromResult handles the final packaging result.
func (m PackageModel) advanceFromResult() (tea.Model, tea.Cmd) {
	msg := m.pendingResult
	m.pendingResult = nil
	m.tickDone = false

	m.done = true
	m.err = msg.Err
	return m, tea.Quit
}

func (m PackageModel) View() string {
	// --- Spinner (in progress) ---
	if !m.done {
		return WrapView(fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), m.phase), m.width)
	}

	// --- Done: determine icon and type badge ---
	specType := ""
	if m.spec != nil {
		specType = m.spec.Type
	}

	typeBadgeStr := ""
	if specType != "" {
		typeBadgeStr = TypeBadge(specType) + "  "
	}

	warnStyle := lipgloss.NewStyle().Foreground(ColorWarning)

	// --- Hard error ---
	if m.err != nil {
		if specType != "" {
			// We know the type — show type badge.
			return WrapView(fmt.Sprintf(" %s  %sFailed to package '%s'\n           %s\n",
				FailIcon(), typeBadgeStr, m.reference, m.err), m.width)
		}
		// No type info — spec couldn't even load.
		label := ""
		if m.reference != "" {
			label = fmt.Sprintf(" '%s'", m.reference)
		}
		return WrapView(fmt.Sprintf(" %s  Failed to package%s\n    %s\n", FailIcon(), label, m.err), m.width)
	}

	// --- Success with warnings ---
	if len(m.warnings) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, " %s  %sSuccessfully packaged taito '%s' with warnings\n",
			SuccessIcon(), typeBadgeStr, m.reference)
		for _, w := range m.warnings {
			b.WriteString(warnStyle.Render("           - "+w) + "\n")
		}
		return WrapView(b.String(), m.width)
	}

	// --- Clean success ---
	return WrapView(fmt.Sprintf(" %s  %sSuccessfully packaged taito '%s'\n",
		SuccessIcon(), typeBadgeStr, m.reference), m.width)
}
