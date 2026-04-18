package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/fsutil"
	"github.com/taito-project/taito/internal/install"
	"github.com/taito-project/taito/internal/spec"
)

func TestInstallSkill(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildSkillDir(t, "gh-skill")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s1, _ := ReadSpec(dir)
	results, err := Install(dir, install.NormalizeReference("github.com/org/repo@v1"), s1, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Name != "gh-skill" {
		t.Errorf("Name = %q, want %q", r.Name, "gh-skill")
	}
	if r.SpecType != spec.TypeSkill {
		t.Errorf("SpecType = %q, want %q", r.SpecType, spec.TypeSkill)
	}
	expectedPath, _ := install.ResolveToolTarget(tool, spec.TypeSkill, "gh-skill", install.NormalizeReference("github.com/org/repo@v1"))
	if r.Path != expectedPath {
		t.Errorf("Path = %q, want %q", r.Path, expectedPath)
	}

	// Verify files were copied.
	if _, err := os.Stat(filepath.Join(expectedPath, "taito.spec")); err != nil {
		t.Errorf("expected taito.spec: %v", err)
	}
	if _, err := os.Stat(filepath.Join(expectedPath, "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md: %v", err)
	}
}

func TestInstallAgent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildAgentDir(t, "gh-agent")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s2, _ := ReadSpec(dir)
	results, err := Install(dir, install.NormalizeReference("github.com/org/repo@v1"), s2, cfg)
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
	expectedPath, _ := install.ResolveToolTarget(tool, spec.TypeAgent, "gh-agent", install.NormalizeReference(install.NormalizeReference("github.com/org/repo@v1")))
	if r.Path != expectedPath {
		t.Errorf("Path = %q, want %q", r.Path, expectedPath)
	}
}

func TestInstallMultipleTools(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildSkillDir(t, "multi-gh")
	tool1, toolRoot1 := fakeTool(t, "tool-a")
	tool2, toolRoot2 := fakeTool(t, "tool-b")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	s3, _ := ReadSpec(dir)
	results, err := Install(dir, "ref", s3, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, root := range []string{toolRoot1, toolRoot2} {
		pDir, _ := install.ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeSkill, "multi-gh", install.NormalizeReference("ref"))
		p := filepath.Join(pDir, "taito.spec")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected taito.spec at %s: %v", p, err)
		}
	}
}

func TestInstallBundle(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildBundleDir(t)
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s4, _ := ReadSpec(dir)
	results, err := Install(dir, install.NormalizeReference("github.com/org/bundle@v1"), s4, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	skillPath, _ := install.ResolveToolTarget(tool, spec.TypeSkill, "helper", install.NormalizeReference("github.com/org/bundle@v1"))
	if _, err := os.Stat(filepath.Join(skillPath, "taito.spec")); err != nil {
		t.Errorf("expected skill taito.spec at %s: %v", skillPath, err)
	}
	if _, err := os.Stat(filepath.Join(skillPath, "SKILL.md")); err != nil {
		t.Errorf("expected skill SKILL.md at %s: %v", skillPath, err)
	}

	agentPath, _ := install.ResolveToolTarget(tool, spec.TypeAgent, "bot", install.NormalizeReference("github.com/org/bundle@v1"))
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

	dir := buildBundleDir(t)
	tool1, toolRoot1 := fakeTool(t, "tool-a")
	tool2, toolRoot2 := fakeTool(t, "tool-b")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	s5, _ := ReadSpec(dir)
	results, err := Install(dir, "ref", s5, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	for _, root := range []string{toolRoot1, toolRoot2} {
		if _, err := os.Stat(filepath.Join(func() string {
			p, _ := install.ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeSkill, "helper", install.NormalizeReference("ref"))
			return p
		}(), "taito.spec")); err != nil {
			t.Errorf("expected helper skill at %s", root)
		}
		if _, err := os.Stat(filepath.Join(func() string {
			p, _ := install.ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, spec.TypeAgent, "bot", install.NormalizeReference("ref"))
			return p
		}(), "taito.spec")); err != nil {
			t.Errorf("expected bot agent at %s", root)
		}
	}
}

func TestInstallNoTools(t *testing.T) {
	dir := buildSkillDir(t, "test")
	cfg := &config.Config{Tools: []config.ToolConfig{}}

	s6, _ := ReadSpec(dir)
	_, err := Install(dir, "ref", s6, cfg)
	if err == nil {
		t.Fatal("expected error for no tools configured")
	}
}

func TestInstallReinstallOverwrites(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildSkillDir(t, "overwrite-gh")
	tool, _ := fakeTool(t, "test-tool")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s7, _ := ReadSpec(dir)
	if _, err := Install(dir, "ref", s7, cfg); err != nil {
		t.Fatalf("Install 1: %v", err)
	}

	pDir, _ := install.ResolveToolTarget(tool, spec.TypeSkill, "overwrite-gh", install.NormalizeReference("ref"))
	sentinelPath := filepath.Join(pDir, "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("should be removed"), 0644); err != nil {
		t.Fatal(err)
	}

	s8, _ := ReadSpec(dir)
	if _, err := Install(dir, "ref", s8, cfg); err != nil {
		t.Fatalf("Install 2: %v", err)
	}

	if _, err := os.Stat(sentinelPath); !os.IsNotExist(err) {
		t.Error("sentinel file should have been removed on reinstall")
	}
	if _, err := os.Stat(filepath.Join(pDir, "taito.spec")); err != nil {
		t.Error("taito.spec should still exist after reinstall")
	}
}

func TestInstallTracksInstalled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := buildSkillDir(t, "tracked-gh")
	tool, _ := fakeTool(t, "cursor")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s9, _ := ReadSpec(dir)
	if _, err := Install(dir, install.NormalizeReference("github.com/org/repo@v1"), s9, cfg); err != nil {
		t.Fatalf("Install: %v", err)
	}

	idx, err := install.LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if (len(idx.Installed.Skills) + len(idx.Installed.Agents)) != 1 {
		t.Fatalf("expected 1 entry, got %d", (len(idx.Installed.Skills) + len(idx.Installed.Agents)))
	}

	e := idx.Installed.Skills[0]
	if e.Name != "tracked-gh" {
		t.Errorf("Name = %q, want %q", e.Name, "tracked-gh")
	}
	if e.Reference != install.NormalizeReference("github.com/org/repo@v1") {
		t.Errorf("Reference = %q", e.Reference)
	}
	if e.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", e.Version, "1.0.0")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(src, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "output")
	if err := fsutil.CopyDir(src, dst, fsutil.CopyDirOptions{CleanTarget: true}); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}
}

func buildSkillDir(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	specData := `{"type":"skill","name":"` + name + `","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "taito.spec"), []byte(specData), 0644); err != nil {
		t.Fatalf("write taito.spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}

func buildAgentDir(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	specData := `{"type":"agent","name":"` + name + `","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "taito.spec"), []byte(specData), 0644); err != nil {
		t.Fatalf("write taito.spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte("# "+name), 0644); err != nil {
		t.Fatalf("write AGENT.md: %v", err)
	}
	return dir
}

func buildBundleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	bundleSpec := `{
  "type": "bundle",
  "name": "test-bundle",
  "version": "1.0.0",
  "includes": [
    "./skills/helper/taito.spec",
    "./agents/bot/taito.spec"
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "taito.spec"), []byte(bundleSpec), 0644); err != nil {
		t.Fatalf("write bundle taito.spec: %v", err)
	}

	skillDir := filepath.Join(dir, "skills", "helper")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "taito.spec"), []byte(`{"type":"skill","name":"helper","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# helper"), 0644); err != nil {
		t.Fatal(err)
	}

	agentDir := filepath.Join(dir, "agents", "bot")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "taito.spec"), []byte(`{"type":"agent","name":"bot","version":"1.0.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "SKILL.md"), []byte("# bot"), 0644); err != nil {
		t.Fatal(err)
	}

	return dir
}

func fakeTool(t *testing.T, name string) (config.ToolConfig, string) {
	t.Helper()
	toolRoot := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(toolRoot, 0755); err != nil {
		t.Fatal(err)
	}
	return config.ToolConfig{Name: name, Path: toolRoot}, toolRoot
}
