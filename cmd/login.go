package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/taito-project/taito/internal/registry"
	"github.com/taito-project/taito/ui"
)

var (
	loginUsername      string
	loginPassword      string
	loginPasswordStdin bool
)

var loginCmd = &cobra.Command{
	Use:   "login <registry>",
	Short: "Log in to an OCI registry",
	Long: `Authenticate to an OCI registry and store credentials for push/pull.

Credentials are verified against the registry before being saved.
If --username and --password (or --password-stdin) are both provided, login
runs non-interactively. Otherwise, an interactive prompt is shown for any
missing values.

Credentials are stored in ~/.config/taito/credentials.json. Native credential
helpers (macOS Keychain, Windows Credential Manager, Linux pass/secretservice)
are used when available.

Examples:
  taito login ghcr.io
  taito login ghcr.io --username myuser --password mytoken
  echo $TOKEN | taito login ghcr.io --username myuser --password-stdin`,
	Args: cobra.ExactArgs(1),
	Run:  runLogin,
}

func init() {
	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Registry username")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Registry password or token")
	loginCmd.Flags().BoolVar(&loginPasswordStdin, "password-stdin", false, "Read password from stdin")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) {
	registryName := args[0]

	// Read password from stdin if requested.
	if loginPasswordStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			loginPassword = strings.TrimRight(scanner.Text(), "\r\n")
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf(" %s  Failed to read password from stdin\n    %v\n", ui.FailIcon(), err)
			os.Exit(1)
		}
	}

	// If both username and password are provided, run non-interactively.
	if loginUsername != "" && loginPassword != "" {
		runLoginNonInteractive(registryName, loginUsername, loginPassword)
		return
	}

	// Interactive mode: launch Bubble Tea TUI.
	runLoginInteractive(registryName, loginUsername)
}

func runLoginNonInteractive(registryName, username, password string) {
	store, err := registry.NewCredentialStore()
	if err != nil {
		fmt.Printf(" %s  Failed to open credential store\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	ctx := context.Background()
	cred := auth.Credential{
		Username: username,
		Password: password,
	}

	if err := registry.Login(ctx, store, registryName, cred); err != nil {
		fmt.Printf(" %s  Login failed for %s\n    %v\n", ui.FailIcon(), registryName, err)
		os.Exit(1)
	}

	fmt.Printf(" %s  Logged in to %s\n", ui.SuccessIcon(), registryName)
}

func runLoginInteractive(registryName, prefillUsername string) {
	loginFn := func(reg, user, pass string) tea.Msg {
		store, err := registry.NewCredentialStore()
		if err != nil {
			return ui.LoginResultMsg{Err: fmt.Errorf("credential store: %w", err)}
		}

		ctx := context.Background()
		cred := auth.Credential{
			Username: user,
			Password: pass,
		}

		if err := registry.Login(ctx, store, reg, cred); err != nil {
			return ui.LoginResultMsg{Err: err}
		}

		return ui.LoginResultMsg{}
	}

	m := ui.NewLoginModel(registryName, prefillUsername, loginFn)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf(" %s  Login failed\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	final, ok := finalModel.(ui.LoginModel)
	if !ok {
		fmt.Printf(" %s  Login failed\n    unexpected model type\n", ui.FailIcon())
		os.Exit(1)
	}

	if final.Cancelled() {
		return
	}

	result := final.Result()
	if result == nil {
		return
	}

	if result.Err != nil {
		// Error was already displayed in the TUI done view.
		os.Exit(1)
	}

	// Success was already displayed in the TUI done view.
}
