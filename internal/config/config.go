// Package config provides cross-platform configuration management for the
// taito CLI. It uses spf13/viper to load a JSON config file from the
// platform-appropriate configuration directory.
//
// Config file locations:
//
//	Linux:   ~/.config/taito/config.json
//	macOS:   ~/.config/taito/config.json  (XDG-style, not ~/Library/Application Support)
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

const appName = "taito"

// Config holds the user's taito configuration.
type Config struct {
	Storage StorageConfig `json:"storage" mapstructure:"storage"`
	Tools   []ToolConfig  `json:"tools,omitempty" mapstructure:"tools"`
}

// StorageConfig holds storage-related settings.
type StorageConfig struct {
	// CacheDir overrides the default cache directory. When empty, the
	// platform default is used (e.g. ~/.cache/taito on Linux).
	CacheDir string `json:"cacheDir,omitempty" mapstructure:"cacheDir"`
}

// ToolConfig represents a configured AI coding tool.
type ToolConfig struct {
	// Name is the tool identifier (e.g. "cursor", "windsurf", "claude-code", "opencode").
	Name string `json:"name" mapstructure:"name"`

	// Path overrides the tool's default config directory. When empty,
	// the well-known default from KnownTools is used.
	Path string `json:"path,omitempty" mapstructure:"path"`
}

// KnownTool describes a supported AI coding tool with its default config path.
type KnownTool struct {
	Name        string // e.g. "cursor"
	DisplayName string // e.g. "Cursor"
	DefaultPath string // absolute platform-specific path (resolved at init time)
}

// knownToolDefs defines the per-platform default paths for each tool.
// On Linux/macOS tools use dotfile directories under $HOME.
// On Windows tools use subdirectories under %APPDATA%.
type knownToolDef struct {
	Name        string
	DisplayName string
	UnixRelPath string // relative to $HOME, e.g. ".cursor"
	WinRelPath  string // relative to %APPDATA%, e.g. "Cursor"
}

var knownToolDefs = []knownToolDef{
	{Name: "cursor", DisplayName: "Cursor", UnixRelPath: ".cursor", WinRelPath: "Cursor"},
	{Name: "windsurf", DisplayName: "Windsurf", UnixRelPath: ".windsurf", WinRelPath: "Windsurf"},
	{Name: "claude-code", DisplayName: "Claude Code", UnixRelPath: ".claude", WinRelPath: "Claude"},
	{Name: "opencode", DisplayName: "OpenCode", UnixRelPath: filepath.Join(".config", "opencode"), WinRelPath: "opencode"},
	{Name: "copilot", DisplayName: "Copilot", UnixRelPath: ".copilot", WinRelPath: "Copilot"},
}

// KnownTools returns the list of supported AI coding tools with their
// platform-resolved default paths. The order determines display order in
// the setup wizard.
func KnownTools() []KnownTool {
	tools := make([]KnownTool, 0, len(knownToolDefs))
	for _, def := range knownToolDefs {
		p, _ := defaultToolPath(def)
		tools = append(tools, KnownTool{
			Name:        def.Name,
			DisplayName: def.DisplayName,
			DefaultPath: p,
		})
	}
	return tools
}

// defaultToolPath resolves the platform-appropriate default path for a tool.
func defaultToolPath(def knownToolDef) (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, def.WinRelPath), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, def.UnixRelPath), nil
}

// ExpandToolPath resolves a path that may start with "~/" to an absolute path.
func ExpandToolPath(p string) (string, error) {
	if len(p) >= 2 && p[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}

// ToolPath returns the effective root path for a configured tool. If the tool
// has a custom Path set, it is returned (with ~ expansion); otherwise the
// platform default from KnownTools() is used.
func (tc ToolConfig) ToolPath() (string, error) {
	if tc.Path != "" {
		return ExpandToolPath(tc.Path)
	}
	for _, kt := range KnownTools() {
		if kt.Name == tc.Name {
			return kt.DefaultPath, nil
		}
	}
	return "", fmt.Errorf("unknown tool: %s", tc.Name)
}

// AgentsDir returns the agents subdirectory for this tool.
func (tc ToolConfig) AgentsDir() (string, error) {
	root, err := tc.ToolPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "agents"), nil
}

// SkillsDir returns the skills subdirectory for this tool.
func (tc ToolConfig) SkillsDir() (string, error) {
	root, err := tc.ToolPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "skills"), nil
}

// Dir returns the path to the taito configuration directory.
//
// On macOS this explicitly returns ~/.config/taito (XDG-style) instead of
// ~/Library/Application Support/taito which os.UserConfigDir() would return.
// On Linux and Windows os.UserConfigDir() already returns the conventional
// location.
func Dir() (string, error) {
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", appName), nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// DefaultCacheDir returns the platform-appropriate default cache directory
// for taito (e.g. ~/.cache/taito on Linux, ~/Library/Caches/taito on macOS,
// %LOCALAPPDATA%\taito on Windows).
func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// Load reads the config file from the standard config directory, applies
// defaults, and returns the parsed Config. If no config file exists, the
// returned Config contains only defaults -- this is not an error.
func Load() (*Config, error) {
	v := viper.New()

	// Defaults.
	v.SetDefault("storage.cacheDir", "")
	v.SetDefault("tools", []ToolConfig{})

	// Config file settings.
	v.SetConfigName("config")
	v.SetConfigType("json")

	configDir, err := Dir()
	if err != nil {
		return nil, err
	}
	v.AddConfigPath(configDir)

	// Read the config file. A missing file is expected on first run.
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// CacheDir returns the effective cache directory. If Config.Storage.CacheDir
// is set it is returned as-is; otherwise the platform default is returned.
func (c *Config) CacheDir() (string, error) {
	if c.Storage.CacheDir != "" {
		return c.Storage.CacheDir, nil
	}
	return DefaultCacheDir()
}

// ResolveTools validates a comma-separated list of tool names and returns the
// corresponding ToolConfig slice. Returns an error if any name is unknown or
// the list is empty.
func ResolveTools(commaSeparated string) ([]ToolConfig, error) {
	knownTools := KnownTools()
	knownMap := make(map[string]bool, len(knownTools))
	var validNames []string
	for _, kt := range knownTools {
		knownMap[kt.Name] = true
		validNames = append(validNames, kt.Name)
	}

	var tools []ToolConfig
	for _, raw := range strings.Split(commaSeparated, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if !knownMap[name] {
			return nil, fmt.Errorf("unknown tool %q; valid tools: %s", name, strings.Join(validNames, ", "))
		}
		tools = append(tools, ToolConfig{Name: name})
	}

	if len(tools) == 0 {
		return nil, fmt.Errorf("no tools specified")
	}
	return tools, nil
}
