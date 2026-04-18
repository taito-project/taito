package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taito-project/taito/internal/config"
)

// Setup wizard steps.
const (
	stepToolSelect = iota // multi-select toggle for AI tools
	stepDone              // terminal state
)

// SetupResult holds the final configuration produced by the setup wizard.
// Read this from the model after the Bubble Tea program exits.
type SetupResult struct {
	Tools []config.ToolConfig
}

// SetupModel is the Bubble Tea model for the "taito setup" command.
type SetupModel struct {
	step int

	// Tool selection.
	tools  []toolItem // one per KnownTool
	cursor int        // currently highlighted tool

	// Terminal state.
	cancelled bool
	warnings  []string // path-existence warnings (populated at finish)
	result    *SetupResult
	err       error
	width     int // terminal width from WindowSizeMsg

	// Styles.
	titleStyle    lipgloss.Style
	subtitleStyle lipgloss.Style
	dimStyle      lipgloss.Style
	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	helpStyle     lipgloss.Style
	warningStyle  lipgloss.Style
}

// toolItem tracks selection state for a known tool.
type toolItem struct {
	tool     config.KnownTool
	selected bool
}

// NewSetupModel creates a SetupModel, optionally pre-filled from an existing config.
func NewSetupModel(existing *config.Config) SetupModel {
	knownTools := config.KnownTools()

	// Build tool items.
	tools := make([]toolItem, len(knownTools))
	for i, kt := range knownTools {
		tools[i] = toolItem{tool: kt}
	}

	// Pre-select tools from existing config.
	if existing != nil {
		for _, tc := range existing.Tools {
			for i := range tools {
				if tools[i].tool.Name == tc.Name {
					tools[i].selected = true
					break
				}
			}
		}
	}

	return SetupModel{
		step:  stepToolSelect,
		tools: tools,

		titleStyle:    lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		subtitleStyle: lipgloss.NewStyle().Foreground(ColorBaseContent).Bold(true),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		cursorStyle:   lipgloss.NewStyle().Foreground(ColorPrimary),
		selectedStyle: lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		helpStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")),
		warningStyle:  lipgloss.NewStyle().Foreground(ColorWarning),
	}
}

// Result returns the final setup result, or nil if cancelled/not done.
func (m SetupModel) Result() *SetupResult {
	return m.result
}

// Cancelled returns true if the user cancelled the wizard.
func (m SetupModel) Cancelled() bool {
	return m.cancelled
}

// Err returns any error that occurred.
func (m SetupModel) Err() error {
	return m.err
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	if m.step == stepToolSelect {
		return m.updateToolSelect(msg)
	}

	return m, nil
}

// --- Tool selection ---

func (m SetupModel) updateToolSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tools)-1 {
				m.cursor++
			}
		case " ":
			m.tools[m.cursor].selected = !m.tools[m.cursor].selected
		case "enter":
			return m.finish()
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// --- Finish: assemble result ---

func (m SetupModel) finish() (tea.Model, tea.Cmd) {
	m.step = stepDone

	var tools []config.ToolConfig
	for _, ti := range m.tools {
		if ti.selected {
			tools = append(tools, config.ToolConfig{Name: ti.tool.Name})
		}
	}

	// Check for path warnings on selected tools.
	m.warnings = nil
	for _, ti := range m.tools {
		if !ti.selected {
			continue
		}
		p := ti.tool.DefaultPath
		if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
			m.warnings = append(m.warnings,
				fmt.Sprintf("%s: path %s does not exist", ti.tool.DisplayName, abbreviateHome(p)))
		}
	}

	m.result = &SetupResult{
		Tools: tools,
	}

	return m, tea.Quit
}

// --- View ---

func (m SetupModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.titleStyle.Render(" taito setup"))
	b.WriteString("\n\n")

	switch m.step {
	case stepToolSelect:
		m.viewToolSelect(&b)
	case stepDone:
		m.viewDone(&b)
	}

	return WrapView(b.String(), m.width)
}

func (m SetupModel) viewToolSelect(b *strings.Builder) {
	b.WriteString(m.subtitleStyle.Render(" AI coding tools"))
	b.WriteString("\n")
	b.WriteString(m.dimStyle.Render(" Used for global skill and agent installation"))
	b.WriteString("\n")
	b.WriteString(m.dimStyle.Render(" Select the tools you use (space to toggle, enter to confirm):"))
	b.WriteString("\n\n")

	for i, item := range m.tools {
		cursor := "  "
		if i == m.cursor {
			cursor = m.cursorStyle.Render("> ")
		}

		check := "[ ]"
		if item.selected {
			check = m.selectedStyle.Render("[x]")
		}

		name := item.tool.DisplayName
		path := m.dimStyle.Render(abbreviateHome(item.tool.DefaultPath))

		fmt.Fprintf(b, " %s%s %s  %s\n", cursor, check, name, path)
	}
	b.WriteString("\n")

	b.WriteString(m.helpStyle.Render(" esc: quit  space: toggle  enter: confirm"))
	b.WriteString("\n\n")
}

func (m SetupModel) viewDone(b *strings.Builder) {
	if m.result == nil {
		return
	}

	knownTools := config.KnownTools()

	fmt.Fprintf(b, " %s  Configuration saved\n\n", SuccessIcon())

	if len(m.result.Tools) > 0 {
		b.WriteString("    Tools:\n")
		for _, tc := range m.result.Tools {
			displayPath := ""
			displayName := tc.Name
			for _, kt := range knownTools {
				if kt.Name == tc.Name {
					displayPath = abbreviateHome(kt.DefaultPath)
					displayName = kt.DisplayName
					break
				}
			}
			fmt.Fprintf(b, "      %-14s %s\n", displayName, displayPath)
			agentsPath := filepath.Join(displayPath, "agents")
			skillsPath := filepath.Join(displayPath, "skills")
			fmt.Fprintf(b, "        agents/    %s\n", agentsPath)
			fmt.Fprintf(b, "        skills/    %s\n", skillsPath)
		}
	} else {
		b.WriteString("    No tools selected.\n")
	}

	// Show path warnings.
	if len(m.warnings) > 0 {
		b.WriteString("\n")
		for _, w := range m.warnings {
			b.WriteString(m.warningStyle.Render("    ! "+w) + "\n")
		}
	}

	// Info about customizing tool paths.
	cfgPath, err := config.ConfigFilePath()
	if err == nil {
		b.WriteString("\n")
		b.WriteString(m.dimStyle.Render(fmt.Sprintf("    To customize tool paths, edit %s", abbreviateHome(cfgPath))))
		b.WriteString("\n")
	}

	b.WriteString("\n")
}

// abbreviateHome replaces the home directory prefix with "~".
func abbreviateHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
