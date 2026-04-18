package ui

import "github.com/charmbracelet/lipgloss"

// Design system color tokens.
// Only colors currently in use are defined here.
// Add more from the palette as needed.
var (
	ColorPrimary     = lipgloss.Color("#ff9f4a")
	ColorAccent      = lipgloss.Color("#ffb151")
	ColorBaseContent = lipgloss.Color("#ffffff")
	ColorInfo        = lipgloss.Color("#00d2ff")
	ColorSuccess     = lipgloss.Color("#0d8085")
	ColorWarning     = lipgloss.Color("#ffcc00")
	ColorError       = lipgloss.Color("#ff716c")
	ColorDim         = lipgloss.Color("#888888")
)

// DimStyle returns a style for secondary / muted text.
func DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorDim)
}

// SuccessIcon renders an orange checkmark.
func SuccessIcon() string {
	return lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("✓")
}

// FailIcon renders a red cross.
func FailIcon() string {
	return lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("✗")
}

// WarningIcon renders a yellow warning indicator.
func WarningIcon() string {
	return lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("⚠")
}

// WrapView applies word-wrapping to the view output so that Bubble Tea's
// renderer does not truncate long lines. If width is 0 (not yet received
// from the terminal), the string is returned as-is.
func WrapView(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

// TypeBadge renders a spec type (agent, skill, bundle) in bold orange text.
func TypeBadge(specType string) string {
	if specType == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(specType)
}
