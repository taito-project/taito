package cmd

import (
	"testing"

	"github.com/taito-project/taito/internal/cache"
)

func TestPushCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "push" {
			found = true
			break
		}
	}
	if !found {
		t.Error("push command not registered on rootCmd")
	}
}

func TestPushCommandMetadata(t *testing.T) {
	if pushCmd.Use != "push <reference>" {
		t.Errorf("Use = %q, want %q", pushCmd.Use, "push <reference>")
	}
	if pushCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestPushCommandRequiresArg(t *testing.T) {
	if pushCmd.Args == nil {
		t.Error("Args should not be nil (expects ExactArgs(1))")
	}
}

func TestPushCommandHasFlags(t *testing.T) {
	f := pushCmd.Flags()

	if f.Lookup("source") == nil {
		t.Error("missing --source flag")
	}

	// Check shorthand.
	if f.ShorthandLookup("s") == nil {
		t.Error("missing -s shorthand for --source")
	}
}

func TestPushCommandNoPasswordFlags(t *testing.T) {
	f := pushCmd.Flags()
	if f.Lookup("username") != nil {
		t.Error("push should not have --username flag (auth is via taito login)")
	}
	if f.Lookup("password") != nil {
		t.Error("push should not have --password flag (auth is via taito login)")
	}
}

func TestPushSourceResolvesFromCache(t *testing.T) {
	// Verify that the cache path uses the sha256 hash of the reference.
	ref := "ghcr.io/org/my-skill:1.0.0"
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	path := cache.PackagePath(cacheDir, ref)
	if path == "" {
		t.Error("expected non-empty path")
	}

	hash := cache.HashReference(ref)
	if !containsAny(path, hash) {
		t.Errorf("path should contain hash %q, got: %s", hash, path)
	}
}

func TestPushDifferentRefsProduceDifferentPaths(t *testing.T) {
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	pathA := cache.PackagePath(cacheDir, "ghcr.io/org-a/test:1.0.0")
	pathB := cache.PackagePath(cacheDir, "ghcr.io/org-b/test:1.0.0")

	if pathA == pathB {
		t.Error("different references should resolve to different cache paths")
	}
}
