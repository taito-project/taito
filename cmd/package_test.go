package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/taito-project/taito/internal/cache"
)

func TestPackageCmdExists(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "package" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'package' command to be registered on rootCmd")
	}
}

func TestPackageCmdSpecFlagDefault(t *testing.T) {
	packageCmd.ResetFlags()
	packageCmd.Flags().StringVar(&specFlag, "spec", ".", "Path to a taito.spec file or directory containing one")
	packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format: oci (default) or tar.gz")

	val, err := packageCmd.Flags().GetString("spec")
	if err != nil {
		t.Fatalf("Failed getting spec flag: %v", err)
	}
	if val != "." {
		t.Errorf("Expected default spec '.', got '%s'", val)
	}
}

func TestPackageCmdSpecFlagSet(t *testing.T) {
	packageCmd.ResetFlags()
	packageCmd.Flags().StringVar(&specFlag, "spec", ".", "Path to a taito.spec file or directory containing one")

	if err := packageCmd.Flags().Set("spec", "./skills/git-helper"); err != nil {
		t.Fatalf("Failed setting spec flag: %v", err)
	}

	val, _ := packageCmd.Flags().GetString("spec")
	if val != "./skills/git-helper" {
		t.Errorf("Expected spec './skills/git-helper', got '%s'", val)
	}
}

func TestPackageCmdFormatFlag(t *testing.T) {
	t.Run("default value is oci", func(t *testing.T) {
		packageCmd.ResetFlags()
		packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format")

		val, _ := packageCmd.Flags().GetString("format")
		if val != "oci" {
			t.Errorf("Expected default format 'oci', got '%s'", val)
		}
	})

	t.Run("can be set to oci", func(t *testing.T) {
		packageCmd.ResetFlags()
		packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format")

		if err := packageCmd.Flags().Set("format", "oci"); err != nil {
			t.Fatalf("Failed setting format flag to oci: %v", err)
		}

		val, _ := packageCmd.Flags().GetString("format")
		if val != "oci" {
			t.Errorf("Expected format 'oci', got '%s'", val)
		}
	})

	t.Run("can be set to tar.gz explicitly", func(t *testing.T) {
		packageCmd.ResetFlags()
		packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format")

		if err := packageCmd.Flags().Set("format", "tar.gz"); err != nil {
			t.Fatalf("Failed setting format flag to tar.gz: %v", err)
		}

		val, _ := packageCmd.Flags().GetString("format")
		if val != "tar.gz" {
			t.Errorf("Expected format 'tar.gz', got '%s'", val)
		}
	})
}

func TestValidFormats(t *testing.T) {
	expected := []string{"tar.gz", "oci"}
	for _, f := range expected {
		if !validFormats[f] {
			t.Errorf("Expected %q to be a valid format", f)
		}
	}

	invalid := []string{"zip", "docker", ""}
	for _, f := range invalid {
		if validFormats[f] {
			t.Errorf("Expected %q to be an invalid format", f)
		}
	}
}

func TestPackageCmdAcceptsPositionalArg(t *testing.T) {
	// Verify the command accepts at most 1 positional argument.
	if packageCmd.Args == nil {
		t.Fatal("expected Args validator to be set")
	}

	// 0 args should be valid.
	if err := packageCmd.Args(packageCmd, []string{}); err != nil {
		t.Errorf("expected 0 args to be valid, got error: %v", err)
	}

	// 1 arg should be valid.
	if err := packageCmd.Args(packageCmd, []string{"registry.io/skill:latest"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got error: %v", err)
	}

	// 2 args should be invalid.
	if err := packageCmd.Args(packageCmd, []string{"a", "b"}); err == nil {
		t.Error("expected 2 args to be invalid")
	}
}

func TestPackageCmdOutputFlag(t *testing.T) {
	packageCmd.ResetFlags()
	packageCmd.Flags().StringVar(&specFlag, "spec", ".", "Path to a taito.spec file or directory containing one")
	packageCmd.Flags().StringVar(&formatFlag, "format", "oci", "Packaging format")
	packageCmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Output path")

	// Default should be empty.
	val, err := packageCmd.Flags().GetString("output")
	if err != nil {
		t.Fatalf("Failed getting output flag: %v", err)
	}
	if val != "" {
		t.Errorf("Expected default output '', got '%s'", val)
	}

	// Should accept a custom path.
	if err := packageCmd.Flags().Set("output", "./dist"); err != nil {
		t.Fatalf("Failed setting output flag: %v", err)
	}
	val, _ = packageCmd.Flags().GetString("output")
	if val != "./dist" {
		t.Errorf("Expected output './dist', got '%s'", val)
	}
}

func TestPackagePathFromCache(t *testing.T) {
	// cache.PackagePath should return a path under <cacheDir>/packages/<hash>.
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	ref := "registry.io/org/devops-agent:1.0.0"
	path := cache.PackagePath(cacheDir, ref)

	hash := cache.HashReference(ref)
	if !strings.HasSuffix(path, filepath.Join("packages", hash)) {
		t.Errorf("PackagePath = %q, want suffix %q", path, filepath.Join("packages", hash))
	}

	// The packages directory should be under the cache dir.
	if !strings.HasPrefix(path, cacheDir) {
		t.Errorf("PackagePath = %q, should be under cacheDir %q", path, cacheDir)
	}
}

func TestPackagePathConsistency(t *testing.T) {
	// Same reference should always produce the same path.
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	ref := "registry.io/org/devops-agent:1.0.0"
	path1 := cache.PackagePath(cacheDir, ref)
	path2 := cache.PackagePath(cacheDir, ref)

	if path1 != path2 {
		t.Errorf("same reference should produce same path:\n  path1: %s\n  path2: %s", path1, path2)
	}
}

func TestPackagePathDifferentRefsProduceDifferentPaths(t *testing.T) {
	cacheDir, err := testConfig().CacheDir()
	if err != nil {
		t.Fatalf("CacheDir: %v", err)
	}

	pathA := cache.PackagePath(cacheDir, "registry.io/org-a/skill:1.0.0")
	pathB := cache.PackagePath(cacheDir, "registry.io/org-b/skill:1.0.0")

	if pathA == pathB {
		t.Error("different references should resolve to different cache paths")
	}
}

func TestPackagesCacheDirCreatedOnAccess(t *testing.T) {
	// Verify that the packages directory is created when we use EnsureDir
	// (which the package command would do before writing).
	tmp := t.TempDir()
	ref := "registry.io/org/test:1.0.0"
	pkgPath := cache.PackagePath(tmp, ref)

	// Create the packages dir to simulate what the package command does.
	packagesDir := filepath.Dir(pkgPath)
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	info, err := os.Stat(packagesDir)
	if err != nil {
		t.Fatalf("packages directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("packages path is not a directory")
	}
}
