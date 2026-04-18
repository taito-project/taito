package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/spec"
)

// buildSkillLayout creates a temp OCI layout containing a single skill.
func buildSkillLayout(t *testing.T, name string) string {
	t.Helper()

	srcDir := t.TempDir()
	specData := `{"type":"skill","name":"` + name + `","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(srcDir, "taito.spec"), []byte(specData), 0644); err != nil {
		t.Fatalf("write taito.spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# "+name), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	layoutDir := filepath.Join(t.TempDir(), name+"-oci")
	s := &spec.TaitoSpec{Type: spec.TypeSkill, Name: name, Version: "1.0.0"}
	if err := archive.CreateOCILayout(srcDir, layoutDir, "latest", s); err != nil {
		t.Fatalf("CreateOCILayout: %v", err)
	}
	return layoutDir
}

// buildAgentLayout creates a temp OCI layout containing a single agent.
func buildAgentLayout(t *testing.T, name string) string {
	t.Helper()

	srcDir := t.TempDir()
	specData := `{"type":"agent","name":"` + name + `","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(srcDir, "taito.spec"), []byte(specData), 0644); err != nil {
		t.Fatalf("write taito.spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# "+name), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	layoutDir := filepath.Join(t.TempDir(), name+"-oci")
	s := &spec.TaitoSpec{Type: spec.TypeAgent, Name: name, Version: "1.0.0"}
	if err := archive.CreateOCILayout(srcDir, layoutDir, "latest", s); err != nil {
		t.Fatalf("CreateOCILayout: %v", err)
	}
	return layoutDir
}

// buildBundleLayout creates a temp OCI layout containing a bundle with child
// skills and agents.
func buildBundleLayout(t *testing.T) string {
	t.Helper()

	srcDir := t.TempDir()

	// Root taito.spec for the bundle.
	bundleSpec := `{
  "type": "bundle",
  "name": "test-bundle",
  "version": "1.0.0",
  "includes": [
    "./skills/helper/taito.spec",
    "./agents/bot/taito.spec"
  ]
}`
	if err := os.WriteFile(filepath.Join(srcDir, "taito.spec"), []byte(bundleSpec), 0644); err != nil {
		t.Fatalf("write bundle taito.spec: %v", err)
	}

	// Child skill.
	skillDir := filepath.Join(srcDir, "skills", "helper")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "taito.spec"), []byte(`{"type":"skill","name":"helper","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# helper"), 0644); err != nil {
		t.Fatal(err)
	}

	// Child agent.
	agentDir := filepath.Join(srcDir, "agents", "bot")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "taito.spec"), []byte(`{"type":"agent","name":"bot","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SKILL.md"), []byte("# bot"), 0644); err != nil {
		t.Fatal(err)
	}

	layoutDir := filepath.Join(t.TempDir(), "bundle-oci")
	s := &spec.TaitoSpec{
		Type:    spec.TypeBundle,
		Name:    "test-bundle",
		Version: "1.0.0",
		Includes: []string{
			"./skills/helper/taito.spec",
			"./agents/bot/taito.spec",
		},
	}
	if err := archive.CreateOCILayout(srcDir, layoutDir, "latest", s); err != nil {
		t.Fatalf("CreateOCILayout: %v", err)
	}
	return layoutDir
}

// fakeTool creates a tool config pointing at a temp directory.
func fakeTool(t *testing.T, name string) (config.ToolConfig, string) {
	t.Helper()
	toolRoot := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(toolRoot, 0755); err != nil {
		t.Fatal(err)
	}
	return config.ToolConfig{Name: name, Path: toolRoot}, toolRoot
}

func TestIsOCILayout(t *testing.T) {
	t.Run("valid layout", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "oci-layout"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
		if !IsOCILayout(dir) {
			t.Error("expected IsOCILayout to return true")
		}
	})

	t.Run("no oci-layout file", func(t *testing.T) {
		dir := t.TempDir()
		if IsOCILayout(dir) {
			t.Error("expected IsOCILayout to return false")
		}
	})

	t.Run("nonexistent path", func(t *testing.T) {
		if IsOCILayout("/nonexistent/path") {
			t.Error("expected IsOCILayout to return false")
		}
	})
}

func TestReadSpecFromLayout(t *testing.T) {
	layoutDir := buildSkillLayout(t, "test-skill")

	s, err := ReadSpecFromLayout(layoutDir)
	if err != nil {
		t.Fatalf("ReadSpecFromLayout: %v", err)
	}
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Type != spec.TypeSkill {
		t.Errorf("Type = %q, want %q", s.Type, spec.TypeSkill)
	}
}

func TestReadSpecFromLayoutAgent(t *testing.T) {
	layoutDir := buildAgentLayout(t, "test-agent")

	s, err := ReadSpecFromLayout(layoutDir)
	if err != nil {
		t.Fatalf("ReadSpecFromLayout: %v", err)
	}
	if s.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", s.Name, "test-agent")
	}
	if s.Type != spec.TypeAgent {
		t.Errorf("Type = %q, want %q", s.Type, spec.TypeAgent)
	}
}

func TestReadSpecFromLayoutBundle(t *testing.T) {
	layoutDir := buildBundleLayout(t)

	s, err := ReadSpecFromLayout(layoutDir)
	if err != nil {
		t.Fatalf("ReadSpecFromLayout: %v", err)
	}
	if s.Name != "test-bundle" {
		t.Errorf("Name = %q, want %q", s.Name, "test-bundle")
	}
	if s.Type != spec.TypeBundle {
		t.Errorf("Type = %q, want %q", s.Type, spec.TypeBundle)
	}
	if len(s.Includes) != 2 {
		t.Errorf("len(Includes) = %d, want 2", len(s.Includes))
	}
}

func TestInstallSingleSkill(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildSkillLayout(t, "my-skill")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s1, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir,  NormalizeReference("ghcr.io/org/my-skill:1.0.0"), s1, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", r.Name, "my-skill")
	}
	if r.SpecType != spec.TypeSkill {
		t.Errorf("SpecType = %q, want %q", r.SpecType, spec.TypeSkill)
	}
	expectedPath, _ := ResolveToolTarget(tool, spec.TypeSkill, "my-skill", NormalizeReference("ghcr.io/org/my-skill:1.0.0"))
	if r.Path != expectedPath {
		t.Errorf("Path = %q, want %q", r.Path, expectedPath)
	}

	// Verify files were extracted.
	specPath := filepath.Join(expectedPath, "taito.spec")
	if _, err := os.Stat(specPath); err != nil {
		t.Errorf("expected taito.spec at %s: %v", specPath, err)
	}
	skillMdPath := filepath.Join(expectedPath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", skillMdPath, err)
	}
}

func TestInstallSingleAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildAgentLayout(t, "my-agent")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s2, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir,  NormalizeReference("ghcr.io/org/my-agent:1.0.0"), s2, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.SpecType != spec.TypeAgent {
		t.Errorf("SpecType = %q, want %q", r.SpecType, spec.TypeAgent)
	}
	expectedPath, _ := ResolveToolTarget(tool, spec.TypeAgent, "my-agent", NormalizeReference("ghcr.io/org/my-agent:1.0.0"))
	if r.Path != expectedPath {
		t.Errorf("Path = %q, want %q", r.Path, expectedPath)
	}
}

func TestInstallMultipleTools(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildSkillLayout(t, "multi-skill")
	tool1, toolRoot1 := fakeTool(t, "tool-a")
	tool2, toolRoot2 := fakeTool(t, "tool-b")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	s3, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir,  "ref", s3, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify both tool directories have the files.
	for _, root := range []string{toolRoot1, toolRoot2} {
		pDir, _ := ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeSkill, "multi-skill", NormalizeReference("ref")); p := filepath.Join(pDir, "taito.spec")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected taito.spec at %s: %v", p, err)
		}
	}
}

func TestInstallBundle(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildBundleLayout(t)
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s4, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir,  NormalizeReference("ghcr.io/org/bundle:1.0.0"), s4, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Bundle has 2 children × 1 tool = 2 results.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify skill was installed.
	skillPath, _ := ResolveToolTarget(tool, spec.TypeSkill, "helper", NormalizeReference("ghcr.io/org/bundle:1.0.0"))
	if _, err := os.Stat(filepath.Join(skillPath, "taito.spec")); err != nil {
		t.Errorf("expected skill taito.spec at %s: %v", skillPath, err)
	}
	if _, err := os.Stat(filepath.Join(skillPath, "SKILL.md")); err != nil {
		t.Errorf("expected skill SKILL.md at %s: %v", skillPath, err)
	}

	// Verify agent was installed.
	agentPath, _ := ResolveToolTarget(tool, spec.TypeAgent, "bot", NormalizeReference("ghcr.io/org/bundle:1.0.0"))
	if _, err := os.Stat(filepath.Join(agentPath, "taito.spec")); err != nil {
		t.Errorf("expected agent taito.spec at %s: %v", agentPath, err)
	}
	if _, err := os.Stat(filepath.Join(agentPath, "SKILL.md")); err != nil {
		t.Errorf("expected agent SKILL.md at %s: %v", agentPath, err)
	}
}

func TestInstallBundleMultipleTools(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildBundleLayout(t)
	tool1, toolRoot1 := fakeTool(t, "tool-a")
	tool2, toolRoot2 := fakeTool(t, "tool-b")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	s5, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir,  "ref", s5, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// 2 children × 2 tools = 4 results.
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify both tools got both children.
	for _, root := range []string{toolRoot1, toolRoot2} {
		if _, err := os.Stat(filepath.Join(func() string { p, _ := ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeSkill, "helper", NormalizeReference("ref")); return p }(), "taito.spec")); err != nil {
			t.Errorf("expected helper skill at %s", root)
		}
		if _, err := os.Stat(filepath.Join(func() string { p, _ := ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeAgent, "bot", NormalizeReference("ref")); return p }(), "taito.spec")); err != nil {
			t.Errorf("expected bot agent at %s", root)
		}
	}
}

func TestInstallNoToolsConfigured(t *testing.T) {
	layoutDir := buildSkillLayout(t, "test")
	cfg := &config.Config{Tools: []config.ToolConfig{}}

	s6, _ := ReadSpecFromLayout(layoutDir)
	_, err := Install(layoutDir,  "ref", s6, cfg)
	if err == nil {
		t.Fatal("expected error for no tools configured")
	}
	if err.Error() != "no tools configured — run 'taito setup' first" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstallReinstallOverwrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildSkillLayout(t, "overwrite-skill")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	// First install.
	s7, _ := ReadSpecFromLayout(layoutDir)
 if _, err := Install(layoutDir,  "ref", s7, cfg); err != nil {
		t.Fatalf("Install 1: %v", err)
	}

	// Write a sentinel file that should be removed on reinstall.
	pDir, _ := ResolveToolTarget(tool, spec.TypeSkill, "overwrite-skill", NormalizeReference("ref")); sentinelPath := filepath.Join(pDir, "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("should be removed"), 0644); err != nil {
		t.Fatal(err)
	}

	// Reinstall.
	s8, _ := ReadSpecFromLayout(layoutDir)
 if _, err := Install(layoutDir,  "ref", s8, cfg); err != nil {
		t.Fatalf("Install 2: %v", err)
	}

	// Sentinel should be gone (directory was cleaned).
	if _, err := os.Stat(sentinelPath); !os.IsNotExist(err) {
		t.Error("sentinel file should have been removed on reinstall")
	}

	// But the real files should still be there.
	if _, err := os.Stat(filepath.Join(pDir, "taito.spec")); err != nil {
		t.Error("taito.spec should still exist after reinstall")
	}
}

func TestInstallTracksInInstalledJson(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	layoutDir := buildSkillLayout(t, "tracked-skill")
	tool, _ := fakeTool(t, "cursor")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s9, _ := ReadSpecFromLayout(layoutDir)
 if _, err := Install(layoutDir,  "ghcr.io/org/tracked:1.0.0", s9, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}

	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if (len(idx.Installed.Skills) + len(idx.Installed.Agents)) != 1 {
		t.Fatalf("expected 1 installed entry, got %d", (len(idx.Installed.Skills) + len(idx.Installed.Agents)))
	}

	e := idx.Installed.Skills[0]
	if e.Name != "tracked-skill" {
		t.Errorf("Name = %q, want %q", e.Name, "tracked-skill")
	}
	if e.Reference != "ghcr.io/org/tracked:1.0.0" {
		t.Errorf("Reference = %q", e.Reference)
	}
	if e.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", e.Version, "1.0.0")
	}
	if e.InstalledAt == "" {
		t.Error("InstalledAt should be set")
	}
}

func TestFindKnownTool(t *testing.T) {
	kt := FindKnownTool("cursor")
	if kt == nil {
		t.Fatal("expected to find cursor")
	}
	if kt.DisplayName != "Cursor" {
		t.Errorf("DisplayName = %q, want %q", kt.DisplayName, "Cursor")
	}

	kt = FindKnownTool("nonexistent")
	if kt != nil {
		t.Error("expected nil for unknown tool")
	}
}

