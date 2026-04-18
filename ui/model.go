package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// A simple model for our Bubble Tea application.
type model struct {
	quitting bool
	width    int // terminal width from WindowSizeMsg
}

// InitialModel initializes the Bubble Tea state model.
func InitialModel() model {
	return model{}
}

// Init initializes the Bubble Tea command, it's called once upon setup.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and state changes.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	// Is it a key press?
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {
		// These keys should exit the program.
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the UI.
func (m model) View() string {
	// If the user hit 'q', 'esc' or 'ctrl+c' quit smoothly.
	if m.quitting {
		return "Goodbye! See you next time.\n"
	}

	// Create a beautiful bordered box using lipgloss.
	var style = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBaseContent).
		Background(ColorPrimary).
		PaddingTop(1).
		PaddingBottom(1).
		PaddingLeft(4).
		PaddingRight(4).
		MarginTop(2).
		MarginBottom(2).
		MarginLeft(2).
		MarginRight(2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent)

	s := style.Render("Welcome to Taito!")
	s += "\n\n  Press 'q' or 'esc' to quit.\n"

	return WrapView(s, m.width)
}
