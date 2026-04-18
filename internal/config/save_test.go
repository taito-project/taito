package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSaveCreatesFile(t *testing.T) {
	tmpHome := t.TempDir()
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	cfg := &Config{
		Storage: StorageConfig{CacheDir: "/tmp/my-cache"},
		Tools: []ToolConfig{
			{Name: "cursor"},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	configDir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Storage.CacheDir != "/tmp/my-cache" {
		t.Errorf("CacheDir = %q, want %q", loaded.Storage.CacheDir, "/tmp/my-cache")
	}
	if len(loaded.Tools) != 1 || loaded.Tools[0].Name != "cursor" {
		t.Errorf("Tools = %+v, want [{Name: cursor}]", loaded.Tools)
	}
}

func TestSaveOverwritesExisting(t *testing.T) {
	tmpHome := t.TempDir()
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	// Write initial config.
	cfg1 := &Config{
		Storage: StorageConfig{CacheDir: "/old"},
	}
	if err := Save(cfg1); err != nil {
		t.Fatalf("Save(cfg1) error: %v", err)
	}

	// Overwrite with new config.
	cfg2 := &Config{
		Storage: StorageConfig{CacheDir: "/new"},
		Tools:   []ToolConfig{{Name: "windsurf", Path: "/custom/windsurf"}},
	}
	if err := Save(cfg2); err != nil {
		t.Fatalf("Save(cfg2) error: %v", err)
	}

	configDir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if loaded.Storage.CacheDir != "/new" {
		t.Errorf("CacheDir = %q, want %q", loaded.Storage.CacheDir, "/new")
	}
	if len(loaded.Tools) != 1 || loaded.Tools[0].Path != "/custom/windsurf" {
		t.Errorf("Tools = %+v, want [{Name: windsurf, Path: /custom/windsurf}]", loaded.Tools)
	}
}

func TestSaveOmitsEmptyFields(t *testing.T) {
	tmpHome := t.TempDir()
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	// A config with empty CacheDir and no tools should not include those fields.
	cfg := &Config{}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	configDir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, "cacheDir") {
		t.Errorf("JSON should omit empty cacheDir, got: %s", raw)
	}
	if strings.Contains(raw, "tools") {
		t.Errorf("JSON should omit empty tools, got: %s", raw)
	}
}

func TestSaveTrailingNewline(t *testing.T) {
	tmpHome := t.TempDir()
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	if err := Save(&Config{}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	configDir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if !strings.HasSuffix(string(data), "\n") {
		t.Error("config.json should end with a trailing newline")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	restoreEnv := overrideHomeDir(t, tmpHome)
	defer restoreEnv()

	cfg := &Config{
		Storage: StorageConfig{CacheDir: "/my/cache"},
		Tools: []ToolConfig{
			{Name: "cursor"},
			{Name: "claude-code", Path: "/opt/claude"},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Storage.CacheDir != cfg.Storage.CacheDir {
		t.Errorf("CacheDir = %q, want %q", loaded.Storage.CacheDir, cfg.Storage.CacheDir)
	}

	if len(loaded.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want 2", len(loaded.Tools))
	}
	if loaded.Tools[0].Name != "cursor" {
		t.Errorf("Tools[0].Name = %q, want %q", loaded.Tools[0].Name, "cursor")
	}
	if loaded.Tools[1].Name != "claude-code" || loaded.Tools[1].Path != "/opt/claude" {
		t.Errorf("Tools[1] = %+v, want {Name: claude-code, Path: /opt/claude}", loaded.Tools[1])
	}
}

func TestConfigFilePath(t *testing.T) {
	p, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath() error: %v", err)
	}
	if filepath.Base(p) != "config.json" {
		t.Errorf("ConfigFilePath() = %q, want base name config.json", p)
	}
	if !filepath.IsAbs(p) {
		t.Errorf("ConfigFilePath() = %q, want absolute path", p)
	}
}

// --- ToolConfig method tests ---

func TestToolPathDefault(t *testing.T) {
	tc := ToolConfig{Name: "cursor"}
	p, err := tc.ToolPath()
	if err != nil {
		t.Fatalf("ToolPath() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cursor")
	if p != want {
		t.Errorf("ToolPath() = %q, want %q", p, want)
	}
}

func TestToolPathOverride(t *testing.T) {
	tc := ToolConfig{Name: "cursor", Path: "/opt/cursor"}
	p, err := tc.ToolPath()
	if err != nil {
		t.Fatalf("ToolPath() error: %v", err)
	}
	if p != "/opt/cursor" {
		t.Errorf("ToolPath() = %q, want %q", p, "/opt/cursor")
	}
}

func TestToolPathTildeOverride(t *testing.T) {
	tc := ToolConfig{Name: "cursor", Path: "~/custom-cursor"}
	p, err := tc.ToolPath()
	if err != nil {
		t.Fatalf("ToolPath() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, "custom-cursor")
	if p != want {
		t.Errorf("ToolPath() = %q, want %q", p, want)
	}
}

func TestToolPathUnknown(t *testing.T) {
	tc := ToolConfig{Name: "unknown-tool"}
	_, err := tc.ToolPath()
	if err == nil {
		t.Fatal("ToolPath() should return error for unknown tool")
	}
}

func TestAgentsDir(t *testing.T) {
	tc := ToolConfig{Name: "cursor"}
	d, err := tc.AgentsDir()
	if err != nil {
		t.Fatalf("AgentsDir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cursor", "agents")
	if d != want {
		t.Errorf("AgentsDir() = %q, want %q", d, want)
	}
}

func TestSkillsDir(t *testing.T) {
	tc := ToolConfig{Name: "opencode"}
	d, err := tc.SkillsDir()
	if err != nil {
		t.Fatalf("SkillsDir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "opencode", "skills")
	if d != want {
		t.Errorf("SkillsDir() = %q, want %q", d, want)
	}
}

func TestExpandToolPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tilde prefix", "~/foo", filepath.Join(home, "foo")},
		{"absolute", "/opt/tool", "/opt/tool"},
		{"relative", "some/path", "some/path"},
		{"tilde only slash", "~/", home},
		{"just tilde no slash", "~", "~"}, // only "~/" is expanded
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandToolPath(tt.in)
			if err != nil {
				t.Fatalf("ExpandToolPath(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ExpandToolPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestKnownToolsHaveDefaults(t *testing.T) {
	tools := KnownTools()
	if len(tools) == 0 {
		t.Fatal("KnownTools() should not be empty")
	}
	for _, kt := range tools {
		if kt.Name == "" {
			t.Error("KnownTool has empty Name")
		}
		if kt.DisplayName == "" {
			t.Errorf("KnownTool %q has empty DisplayName", kt.Name)
		}
		if kt.DefaultPath == "" {
			t.Errorf("KnownTool %q has empty DefaultPath", kt.Name)
		}
		if !filepath.IsAbs(kt.DefaultPath) {
			t.Errorf("KnownTool %q DefaultPath %q should be absolute", kt.Name, kt.DefaultPath)
		}
	}
}

func TestLoadWithTools(t *testing.T) {
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

	cfgData := map[string]interface{}{
		"tools": []map[string]string{
			{"name": "cursor"},
			{"name": "claude-code", "path": "/opt/claude"},
		},
	}
	data, _ := json.Marshal(cfgData)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), data, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(cfg.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want 2", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "cursor" {
		t.Errorf("Tools[0].Name = %q, want %q", cfg.Tools[0].Name, "cursor")
	}
	if cfg.Tools[1].Path != "/opt/claude" {
		t.Errorf("Tools[1].Path = %q, want %q", cfg.Tools[1].Path, "/opt/claude")
	}
}

// overrideHomeDir is defined in config_test.go — this file shares the package.
// The function is available via the test binary, no need to redefine it.
// However, if running this file's tests independently fails, ensure config_test.go
// is also compiled. Since they share the same package, Go always compiles all
// _test.go files together, so this is guaranteed.

// Note: overrideHomeDir uses runtime.GOOS so tests adapt to the platform.
var _ = runtime.GOOS // suppress unused import if needed
