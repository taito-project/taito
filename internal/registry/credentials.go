// Package registry provides OCI registry credential management for the
// taito CLI. Credentials are stored in ~/.config/taito/credentials.json
// using the ORAS credentials library, which delegates to native credential
// helpers (macOS Keychain, Windows Credential Manager, Linux pass/secretservice)
// when available and falls back to plaintext JSON when not.
package registry

import (
	"context"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"

	"github.com/taito-project/taito/internal/config"
)

const credentialFileName = "credentials.json"

// CredentialFilePath returns the full path to the credentials file.
func CredentialFilePath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, credentialFileName), nil
}

// NewCredentialStore returns a DynamicStore backed by the taito credentials
// file. It uses native credential helpers when available and falls back to
// plaintext JSON storage.
func NewCredentialStore() (*credentials.DynamicStore, error) {
	path, err := CredentialFilePath()
	if err != nil {
		return nil, err
	}
	return credentials.NewStore(path, credentials.StoreOptions{
		AllowPlaintextPut:        true,
		DetectDefaultNativeStore: true,
	})
}

// Login verifies the given credentials against the named registry and, on
// success, stores them in the credential store. The registry name should be
// a hostname (e.g. "ghcr.io", "registry.example.com").
func Login(ctx context.Context, store credentials.Store, registryName string, cred auth.Credential) error {
	reg, err := remote.NewRegistry(registryName)
	if err != nil {
		return err
	}
	return credentials.Login(ctx, store, reg, cred)
}

// Logout removes credentials for the given registry from the credential store.
func Logout(ctx context.Context, store credentials.Store, registryName string) error {
	return credentials.Logout(ctx, store, registryName)
}

// GetCredential retrieves the stored credential for the given registry.
// Returns auth.EmptyCredential if no credential is stored.
func GetCredential(ctx context.Context, store credentials.Store, registryName string) (auth.Credential, error) {
	serverAddr := credentials.ServerAddressFromHostname(registryName)
	return store.Get(ctx, serverAddr)
}
