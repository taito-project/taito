package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/taito-project/taito/internal/registry"
	"github.com/taito-project/taito/ui"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <registry>",
	Short: "Log out of an OCI registry",
	Long: `Remove stored credentials for an OCI registry.

Examples:
  taito logout ghcr.io
  taito logout registry.example.com`,
	Args: cobra.ExactArgs(1),
	Run:  runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) {
	registryName := args[0]

	store, err := registry.NewCredentialStore()
	if err != nil {
		fmt.Printf(" %s  Failed to open credential store\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Check if credentials exist before attempting to remove them.
	cred, err := registry.GetCredential(ctx, store, registryName)
	if err != nil {
		fmt.Printf(" %s  Failed to read credentials\n    %v\n", ui.FailIcon(), err)
		os.Exit(1)
	}

	if cred == auth.EmptyCredential {
		fmt.Printf(" %s  Not logged in to %s\n", ui.FailIcon(), registryName)
		return
	}

	if err := registry.Logout(ctx, store, registryName); err != nil {
		fmt.Printf(" %s  Failed to log out of %s\n    %v\n", ui.FailIcon(), registryName, err)
		os.Exit(1)
	}

	fmt.Printf(" %s  Logged out of %s\n", ui.SuccessIcon(), registryName)
}
