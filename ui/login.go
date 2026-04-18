package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Login wizard steps.
const (
	loginStepUsername  = iota // text input for username
	loginStepPassword         // text input for password (masked)
	loginStepVerifying        // spinner while verifying credentials
	loginStepDone             // terminal state
)

// loginMinSpinnerDuration is the minimum time the verifying spinner is shown.
const loginMinSpinnerDuration = 300 * time.Millisecond

// LoginResultMsg is sent when the async login attempt completes.
type LoginResultMsg struct {
	Err error
}

// loginTickMsg signals that the minimum spinner duration has elapsed.
type loginTickMsg struct{}

// LoginResult holds the final outcome of the login wizard.
type LoginResult struct {
	Registry string
	Username string
	Password string
	Err      error
}

// LoginModel is the Bubble Tea model for the "taito login" command.
// It prompts for username and password (skipping fields pre-filled from flags),
// then shows a spinner while verifying credentials.
type LoginModel struct {
	step     int
	registry string // always provided from positional arg

	// Text inputs.
	usernameInput textinput.Model
	passwordInput textinput.Model

	// Spinner for verification phase.
	spinner spinner.Model

	// loginFn is the async function that performs credential verification.
	// Injected by the caller. It receives (registry, username, password) and
	// returns a LoginResultMsg.
	loginFn func(registry, username, password string) tea.Msg

	// Phase gating for spinner minimum duration.
	tickDone      bool
	pendingResult *LoginResultMsg

	// Terminal state.
	cancelled bool
	result    *LoginResult
	err       error
	width     int // terminal width from WindowSizeMsg

	// Styles.
	titleStyle    lipgloss.Style
	subtitleStyle lipgloss.Style
	dimStyle      lipgloss.Style
	helpStyle     lipgloss.Style
}

// NewLoginModel creates a LoginModel.
//
// registry is required (from positional arg).
// username may be pre-filled from --username flag (empty string = prompt for it).
// loginFn performs the actual credential verification and is called asynchronously.
func NewLoginModel(registry, username string, loginFn func(registry, username, password string) tea.Msg) LoginModel {
	// Username input.
	ui := textinput.New()
	ui.Placeholder = "username"
	ui.CharLimit = 256
	ui.Width = 40
	if username != "" {
		ui.SetValue(username)
	}

	// Password input.
	pi := textinput.New()
	pi.Placeholder = "password or token"
	pi.CharLimit = 512
	pi.Width = 40
	pi.EchoMode = textinput.EchoPassword
	pi.EchoCharacter = '*'

	// Spinner.
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	// Determine starting step and focus the appropriate input.
	startStep := loginStepUsername
	if username != "" {
		startStep = loginStepPassword
		pi.Focus()
	} else {
		ui.Focus()
	}

	return LoginModel{
		step:          startStep,
		registry:      registry,
		usernameInput: ui,
		passwordInput: pi,
		spinner:       s,
		loginFn:       loginFn,

		titleStyle:    lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		subtitleStyle: lipgloss.NewStyle().Foreground(ColorBaseContent).Bold(true),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		helpStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")),
	}
}

// Result returns the final login result, or nil if cancelled/not done.
func (m LoginModel) Result() *LoginResult {
	return m.result
}

// Cancelled returns true if the user cancelled the wizard.
func (m LoginModel) Cancelled() bool {
	return m.cancelled
}

// Err returns any error that occurred.
func (m LoginModel) Err() error {
	return m.err
}

func (m LoginModel) Init() tea.Cmd {
	// Focus is set in the constructor. Init only needs to start the cursor blink.
	switch m.step {
	case loginStepUsername:
		return m.usernameInput.Cursor.BlinkCmd()
	case loginStepPassword:
		return m.passwordInput.Cursor.BlinkCmd()
	}
	return nil
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	switch m.step {
	case loginStepUsername:
		return m.updateUsername(msg)
	case loginStepPassword:
		return m.updatePassword(msg)
	case loginStepVerifying:
		return m.updateVerifying(msg)
	}

	return m, nil
}

// --- Username step ---

func (m LoginModel) updateUsername(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.usernameInput.Value())
			if val == "" {
				return m, nil // don't advance on empty
			}
			m.step = loginStepPassword
			m.usernameInput.Blur()
			m.passwordInput.Focus()
			return m, m.passwordInput.Cursor.BlinkCmd()
		}
	}

	var cmd tea.Cmd
	m.usernameInput, cmd = m.usernameInput.Update(msg)
	return m, cmd
}

// --- Password step ---

func (m LoginModel) updatePassword(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.passwordInput.Value()
			if val == "" {
				return m, nil // don't advance on empty
			}
			return m.startVerifying()
		}
	}

	var cmd tea.Cmd
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

// --- Verifying step ---

func (m LoginModel) startVerifying() (tea.Model, tea.Cmd) {
	m.step = loginStepVerifying
	m.passwordInput.Blur()
	m.tickDone = false
	m.pendingResult = nil

	reg := m.registry
	user := strings.TrimSpace(m.usernameInput.Value())
	pass := m.passwordInput.Value()
	loginFn := m.loginFn

	return m, tea.Batch(
		m.spinner.Tick,
		tea.Tick(loginMinSpinnerDuration, func(time.Time) tea.Msg { return loginTickMsg{} }),
		func() tea.Msg {
			return loginFn(reg, user, pass)
		},
	)
}

func (m LoginModel) updateVerifying(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loginTickMsg:
		m.tickDone = true
		return m.tryFinish()

	case LoginResultMsg:
		m.pendingResult = &msg
		return m.tryFinish()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// tryFinish checks whether both the tick and login result have arrived.
func (m LoginModel) tryFinish() (tea.Model, tea.Cmd) {
	if !m.tickDone || m.pendingResult == nil {
		return m, nil
	}

	m.step = loginStepDone

	username := strings.TrimSpace(m.usernameInput.Value())
	password := m.passwordInput.Value()

	m.result = &LoginResult{
		Registry: m.registry,
		Username: username,
		Password: password,
		Err:      m.pendingResult.Err,
	}
	m.err = m.pendingResult.Err

	return m, tea.Quit
}

// --- View ---

func (m LoginModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.titleStyle.Render(" taito login"))
	b.WriteString("\n\n")

	switch m.step {
	case loginStepUsername:
		m.viewUsername(&b)
	case loginStepPassword:
		m.viewPassword(&b)
	case loginStepVerifying:
		m.viewVerifying(&b)
	case loginStepDone:
		m.viewDone(&b)
	}

	return WrapView(b.String(), m.width)
}

func (m LoginModel) viewUsername(b *strings.Builder) {
	b.WriteString(m.subtitleStyle.Render(fmt.Sprintf(" Log in to %s", m.registry)))
	b.WriteString("\n\n")
	fmt.Fprintf(b, "   Username: %s\n", m.usernameInput.View())
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render(" enter: continue  esc: cancel"))
	b.WriteString("\n\n")
}

func (m LoginModel) viewPassword(b *strings.Builder) {
	b.WriteString(m.subtitleStyle.Render(fmt.Sprintf(" Log in to %s", m.registry)))
	b.WriteString("\n\n")

	username := strings.TrimSpace(m.usernameInput.Value())
	fmt.Fprintf(b, "   Username: %s\n", m.dimStyle.Render(username))
	fmt.Fprintf(b, "   Password: %s\n", m.passwordInput.View())
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render(" enter: log in  esc: cancel"))
	b.WriteString("\n\n")
}

func (m LoginModel) viewVerifying(b *strings.Builder) {
	fmt.Fprintf(b, " %s Logging in to %s...\n\n", m.spinner.View(), m.registry)
}

func (m LoginModel) viewDone(b *strings.Builder) {
	if m.result == nil {
		return
	}

	if m.result.Err != nil {
		fmt.Fprintf(b, " %s  Login failed for %s\n", FailIcon(), m.registry)
		fmt.Fprintf(b, "    %v\n\n", m.result.Err)
		return
	}

	fmt.Fprintf(b, " %s  Logged in to %s\n\n", SuccessIcon(), m.registry)
}
