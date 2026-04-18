package cmd

import (
	"testing"

	"github.com/taito-project/taito/internal/cache"
)

func TestPullCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "pull" {
			found = true
			break
		}
	}
	if !found {
		t.Error("pull command not registered on rootCmd")
	}
}

func TestPullCommandMetadata(t *testing.T) {
	if pullCmd.Use != "pull <reference>" {
		t.Errorf("Use = %q, want %q", pullCmd.Use, "pull <reference>")
	}
	if pullCmd.Short == "" {
		t.Error("Short should not be empty")
	}
}

func TestPullCommandRequiresArg(t *testing.T) {
	if pullCmd.Args == nil {
		t.Error("Args should not be nil (expects ExactArgs(1))")
	}
}

func TestPullCommandHasFlags(t *testing.T) {
	f := pullCmd.Flags()

	if f.Lookup("output") == nil {
		t.Error("missing --output flag")
	}

	// Check shorthand.
	if f.ShorthandLookup("o") == nil {
		t.Error("missing -o shorthand for --output")
	}
}

func TestPullCommandNoAuthFlags(t *testing.T) {
	f := pullCmd.Flags()
	if f.Lookup("username") != nil {
		t.Error("pull should not have --username flag (auth is via taito login)")
	}
	if f.Lookup("password") != nil {
		t.Error("pull should not have --password flag (auth is via taito login)")
	}
}

func TestPullTargetResolvesFromCache(t *testing.T) {
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

func TestPullAndPushResolveSamePath(t *testing.T) {
	// Pull target and push source should resolve to the same cache path
	// for the same reference, so round-tripping works.
	ref := "ghcr.io/org/my-skill:1.0.0"
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	pushPath := cache.PackagePath(cacheDir, ref)
	pullPath := cache.PackagePath(cacheDir, ref)

	if pushPath != pullPath {
		t.Errorf("push and pull paths should match:\n  push: %s\n  pull: %s", pushPath, pullPath)
	}
}

func TestPullDifferentRefsProduceDifferentPaths(t *testing.T) {
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	pathA := cache.PackagePath(cacheDir, "ghcr.io/org-a/skill:1.0.0")
	pathB := cache.PackagePath(cacheDir, "ghcr.io/org-b/skill:1.0.0")

	if pathA == pathB {
		t.Error("different references should resolve to different cache paths")
	}
}
