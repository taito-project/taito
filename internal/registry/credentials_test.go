package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// newTestFileStore creates a plaintext FileStore in a temp directory.
// Returns the store and a cleanup function.
func newTestFileStore(t *testing.T) (*credentials.FileStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	store, err := credentials.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	return store, path
}

func TestCredentialFilePath(t *testing.T) {
	path, err := CredentialFilePath()
	if err != nil {
		t.Fatalf("CredentialFilePath: %v", err)
	}
	if filepath.Base(path) != "credentials.json" {
		t.Errorf("expected basename credentials.json, got %s", filepath.Base(path))
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %s", path)
	}
}

func TestNewCredentialStore(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("NewCredentialStore: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestPutAndGetCredential(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	cred := auth.Credential{
		Username: "testuser",
		Password: "testpass",
	}

	err := store.Put(ctx, "registry.example.com", cred)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := store.Get(ctx, "registry.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Username != "testuser" {
		t.Errorf("Username = %q, want %q", got.Username, "testuser")
	}
	if got.Password != "testpass" {
		t.Errorf("Password = %q, want %q", got.Password, "testpass")
	}
}

func TestGetCredentialNotFound(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	got, err := store.Get(ctx, "nonexistent.example.com")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Should return empty credential.
	if got != auth.EmptyCredential {
		t.Errorf("expected EmptyCredential, got %+v", got)
	}
}

func TestDeleteCredential(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	cred := auth.Credential{
		Username: "user",
		Password: "pass",
	}

	if err := store.Put(ctx, "registry.example.com", cred); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := store.Delete(ctx, "registry.example.com"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := store.Get(ctx, "registry.example.com")
	if err != nil {
		t.Fatalf("Get after Delete: %v", err)
	}
	if got != auth.EmptyCredential {
		t.Errorf("expected EmptyCredential after delete, got %+v", got)
	}
}

func TestLogoutRemovesCredential(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	cred := auth.Credential{
		Username: "user",
		Password: "pass",
	}

	if err := store.Put(ctx, "registry.example.com", cred); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if err := Logout(ctx, store, "registry.example.com"); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	got, err := store.Get(ctx, "registry.example.com")
	if err != nil {
		t.Fatalf("Get after Logout: %v", err)
	}
	if got != auth.EmptyCredential {
		t.Errorf("expected EmptyCredential after Logout, got %+v", got)
	}
}

func TestLogoutNonExistentIsNotError(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	// Logout of a registry we never logged into should not error.
	err := Logout(ctx, store, "never-logged-in.example.com")
	if err != nil {
		t.Errorf("Logout of non-existent registry should not error, got: %v", err)
	}
}

func TestGetCredentialHelper(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	cred := auth.Credential{
		Username: "user",
		Password: "token123",
	}

	if err := store.Put(ctx, "ghcr.io", cred); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := GetCredential(ctx, store, "ghcr.io")
	if err != nil {
		t.Fatalf("GetCredential: %v", err)
	}
	if got.Username != "user" {
		t.Errorf("Username = %q, want %q", got.Username, "user")
	}
	if got.Password != "token123" {
		t.Errorf("Password = %q, want %q", got.Password, "token123")
	}
}

func TestMultipleRegistries(t *testing.T) {
	store, _ := newTestFileStore(t)
	ctx := context.Background()

	registries := map[string]auth.Credential{
		"ghcr.io":              {Username: "gh-user", Password: "gh-token"},
		"registry.example.com": {Username: "ex-user", Password: "ex-pass"},
		"docker.io":            {Username: "dk-user", Password: "dk-pass"},
	}

	for reg, cred := range registries {
		if err := store.Put(ctx, reg, cred); err != nil {
			t.Fatalf("Put(%s): %v", reg, err)
		}
	}

	for reg, want := range registries {
		got, err := store.Get(ctx, reg)
		if err != nil {
			t.Fatalf("Get(%s): %v", reg, err)
		}
		if got.Username != want.Username {
			t.Errorf("Get(%s).Username = %q, want %q", reg, got.Username, want.Username)
		}
		if got.Password != want.Password {
			t.Errorf("Get(%s).Password = %q, want %q", reg, got.Password, want.Password)
		}
	}
}

func TestCredentialFilePersisted(t *testing.T) {
	store, path := newTestFileStore(t)
	ctx := context.Background()

	cred := auth.Credential{
		Username: "persist-user",
		Password: "persist-pass",
	}

	if err := store.Put(ctx, "registry.example.com", cred); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Verify the file was created.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("credentials file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("credentials file is empty")
	}

	// Open a new store from the same file and verify persistence.
	store2, err := credentials.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore (reopen): %v", err)
	}

	got, err := store2.Get(ctx, "registry.example.com")
	if err != nil {
		t.Fatalf("Get from reopened store: %v", err)
	}
	if got.Username != "persist-user" {
		t.Errorf("Username = %q, want %q", got.Username, "persist-user")
	}
}
