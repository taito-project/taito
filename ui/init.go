package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/taito-project/taito/internal/spec"
)

// Init wizard steps
const (
	initStepType = iota
	initStepName
	initStepDescription
	initStepDone
)

type InitResult struct {
	Spec *spec.TaitoSpec
}

type InitModel struct {
	step      int
	cancelled bool
	result    *InitResult
	err       error
	width     int

	// Type selection
	types    []string
	cursor   int
	specType string

	// Text inputs
	nameInput textinput.Model
	descInput textinput.Model

	namePattern *regexp.Regexp

	titleStyle    lipgloss.Style
	subtitleStyle lipgloss.Style
	dimStyle      lipgloss.Style
	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	helpStyle     lipgloss.Style
	errorStyle    lipgloss.Style
}

func NewInitModel() InitModel {
	types := []string{spec.TypeSkill, spec.TypeAgent, spec.TypeBundle}

	nameInput := textinput.New()
	nameInput.Placeholder = "my-skill"
	nameInput.CharLimit = 128
	nameInput.Width = 40

	descInput := textinput.New()
	descInput.Placeholder = "A brief description (optional)"
	descInput.CharLimit = 500
	descInput.Width = 60

	return InitModel{
		step:        initStepType,
		types:       types,
		nameInput:   nameInput,
		descInput:   descInput,
		namePattern: regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`),

		titleStyle:    lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		subtitleStyle: lipgloss.NewStyle().Foreground(ColorBaseContent).Bold(true),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		cursorStyle:   lipgloss.NewStyle().Foreground(ColorPrimary),
		selectedStyle: lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		helpStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")),
		errorStyle:    lipgloss.NewStyle().Foreground(ColorError),
	}
}

func (m InitModel) Result() *InitResult {
	return m.result
}

func (m InitModel) Cancelled() bool {
	return m.cancelled
}

func (m InitModel) Err() error {
	return m.err
}

func (m InitModel) Init() tea.Cmd {
	return nil
}

func (m InitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	switch m.step {
	case initStepType:
		return m.updateType(msg)
	case initStepName:
		return m.updateName(msg)
	case initStepDescription:
		return m.updateDescription(msg)
	}

	return m, nil
}

func (m InitModel) updateType(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.types) - 1
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.types) {
				m.cursor = 0
			}
		case "enter":
			m.specType = m.types[m.cursor]
			m.step = initStepName
			m.nameInput.Focus()
			return m, m.nameInput.Cursor.BlinkCmd()
		}
	}
	return m, nil
}

func (m InitModel) updateName(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.nameInput.Value())
			if m.namePattern.MatchString(val) {
				m.step = initStepDescription
				m.nameInput.Blur()
				m.descInput.Focus()
				return m, m.descInput.Cursor.BlinkCmd()
			}
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m InitModel) updateDescription(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.descInput.Value())

			s := &spec.TaitoSpec{
				Type: m.specType,
				Name: strings.TrimSpace(m.nameInput.Value()),
			}
			if val != "" {
				s.Description = val
			}
			if m.specType == spec.TypeBundle {
				s.Includes = []string{"<path ref to your skills / agents>"}
			}

			m.result = &InitResult{Spec: s}
			m.step = initStepDone
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func (m InitModel) View() string {
	if m.cancelled || m.step == initStepDone {
		return ""
	}

	var b strings.Builder
	b.WriteString(m.titleStyle.Render("Initialize a new taito.spec"))
	b.WriteString("\n\n")

	switch m.step {
	case initStepType:
		b.WriteString("Select package type:\n\n")
		for i, t := range m.types {
			cursor := "  "
			if m.cursor == i {
				cursor = m.cursorStyle.Render("> ")
				b.WriteString(cursor + m.selectedStyle.Render(t) + "\n")
			} else {
				b.WriteString(cursor + t + "\n")
			}
		}
		b.WriteString("\n" + m.helpStyle.Render("↑/↓: navigate  enter: select"))

	case initStepName:
		fmt.Fprintf(&b, "Type: %s\n\n", m.dimStyle.Render(m.specType))
		b.WriteString("Package name (e.g. my-skill):\n")
		b.WriteString(m.nameInput.View() + "\n")

		val := strings.TrimSpace(m.nameInput.Value())
		if val != "" && !m.namePattern.MatchString(val) {
			b.WriteString("\n" + m.errorStyle.Render("✗ Name must match ^[a-z0-9][a-z0-9_-]*$"))
		}
		b.WriteString("\n\n" + m.helpStyle.Render("enter: confirm"))

	case initStepDescription:
		fmt.Fprintf(&b, "Type: %s\n", m.dimStyle.Render(m.specType))
		fmt.Fprintf(&b, "Name: %s\n\n", m.dimStyle.Render(strings.TrimSpace(m.nameInput.Value())))
		b.WriteString("Description (optional):\n")
		b.WriteString(m.descInput.View() + "\n")
		b.WriteString("\n\n" + m.helpStyle.Render("enter: finish"))
	}

	return b.String()
}
