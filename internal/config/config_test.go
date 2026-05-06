package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDirEndsWithTaito(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if filepath.Base(dir) != "taito" {
		t.Errorf("Dir() = %q, want base name %q", dir, "taito")
	}
}

func TestDirIsAbsolute(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("Dir() = %q, want absolute path", dir)
	}
}

func TestDirDarwinUsesXDG(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	// On macOS we explicitly want ~/.config/taito, NOT ~/Library/Application Support/taito.
	if strings.Contains(dir, "Library") {
		t.Errorf("Dir() on macOS = %q, should use XDG-style ~/.config, not Library", dir)
	}
	if !strings.Contains(dir, ".config") {
		t.Errorf("Dir() on macOS = %q, expected to contain .config", dir)
	}
}

func TestDefaultCacheDirEndsWithTaito(t *testing.T) {
	dir, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("DefaultCacheDir() error: %v", err)
	}
	if filepath.Base(dir) != "taito" {
		t.Errorf("DefaultCacheDir() = %q, want base name %q", dir, "taito")
	}
}

func TestDefaultCacheDirIsAbsolute(t *testing.T) {
	dir, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("DefaultCacheDir() error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("DefaultCacheDir() = %q, want absolute path", dir)
	}
}

func TestLoadMissingFile(t *testing.T) {
	// Load should succeed even when no config file exists by using defaults.
	// We point HOME at a temp dir to guarantee no config file is found.
	tmpHome := t.TempDir()

	// Save and restore env vars.
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.Storage.CacheDir != "" {
		t.Errorf("Storage.CacheDir = %q, want empty (default)", cfg.Storage.CacheDir)
	}
}

func TestLoadWithCacheDir(t *testing.T) {
	tmpHome := t.TempDir()

	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	// Create the config directory and write a config.json.
	configDir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	customCache := "/tmp/my-taito-cache"
	cfgData := map[string]interface{}{
		"storage": map[string]interface{}{
			"cacheDir": customCache,
		},
	}
	data, err := json.Marshal(cfgData)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Storage.CacheDir != customCache {
		t.Errorf("Storage.CacheDir = %q, want %q", cfg.Storage.CacheDir, customCache)
	}
}

func TestCacheDirDefault(t *testing.T) {
	// When CacheDir is empty, CacheDir() should return the platform default.
	cfg := &Config{}
	dir, err := cfg.CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error: %v", err)
	}

	defaultDir, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("DefaultCacheDir() error: %v", err)
	}
	if dir != defaultDir {
		t.Errorf("CacheDir() = %q, want default %q", dir, defaultDir)
	}
}

func TestCacheDirOverride(t *testing.T) {
	custom := "/opt/taito/cache"
	cfg := &Config{
		Storage: StorageConfig{CacheDir: custom},
	}
	dir, err := cfg.CacheDir()
	if err != nil {
		t.Fatalf("CacheDir() error: %v", err)
	}
	if dir != custom {
		t.Errorf("CacheDir() = %q, want %q", dir, custom)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpHome := t.TempDir()

	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	configDir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	// Write invalid JSON.
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	tmpHome := t.TempDir()

	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	configDir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	// Write an empty JSON object.
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Storage.CacheDir != "" {
		t.Errorf("Storage.CacheDir = %q, want empty", cfg.Storage.CacheDir)
	}
}

// overrideHomeDir temporarily sets the HOME (or USERPROFILE on Windows) and
// XDG/APPDATA environment variables so that Dir() and Load() resolve to a
// temp directory. Returns a restore function.
func overrideHomeDir(t *testing.T, tmpHome string) func() {
	t.Helper()

	saves := make(map[string]string)
	setEnv := func(key, val string) {
		saves[key] = os.Getenv(key)
		_ = os.Setenv(key, val)
	}

	switch runtime.GOOS {
	case "windows":
		setEnv("USERPROFILE", tmpHome)
		setEnv("APPDATA", filepath.Join(tmpHome, "AppData", "Roaming"))
		setEnv("LOCALAPPDATA", filepath.Join(tmpHome, "AppData", "Local"))
	case "darwin":
		setEnv("HOME", tmpHome)
	default: // linux, freebsd, etc.
		setEnv("HOME", tmpHome)
		setEnv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))
		setEnv("XDG_CACHE_HOME", filepath.Join(tmpHome, ".cache"))
	}

	return func() {
		for k, v := range saves {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}
}

func TestResolveTools_Valid(t *testing.T) {
	tools, err := ResolveTools("opencode,claude-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "opencode" || tools[1].Name != "claude-code" {
		t.Errorf("unexpected tools: %v", tools)
	}
}

func TestResolveTools_InvalidName(t *testing.T) {
	_, err := ResolveTools("opencode,fakeTool")
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "fakeTool") {
		t.Errorf("error should mention invalid name, got: %v", err)
	}
}

func TestResolveTools_Empty(t *testing.T) {
	_, err := ResolveTools("")
	if err == nil {
		t.Fatal("expected error for empty tools")
	}
}

func TestResolveTools_WhitespaceHandling(t *testing.T) {
	tools, err := ResolveTools(" opencode , claude-code ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}
