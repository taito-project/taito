package install

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/taito-project/taito/internal/archive"
	"github.com/taito-project/taito/internal/config"
	"github.com/taito-project/taito/internal/spec"
)

// projectRoot returns the absolute path to the project root by walking up from
// the test file's location.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	// filename is internal/install/integration_test.go → project root is two levels up
	return filepath.Join(filepath.Dir(filename), "..", "..")
}

// bundleRepoDir returns the path to the example bundle-repo.
func bundleRepoDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(projectRoot(t), "taito.spec", "examples", "bundle-repo")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bundle-repo examples not found at %s: %v", dir, err)
	}
	return dir
}

// singleSkillDir returns the path to the example git-commit-helper skill.
func singleSkillDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(bundleRepoDir(t), "skills", "git-commit-helper")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("git-commit-helper skill not found at %s: %v", dir, err)
	}
	return dir
}

// singleAgentDir returns the path to the example devops-agent.
func singleAgentDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(bundleRepoDir(t), "agents", "devops-agent")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("devops-agent not found at %s: %v", dir, err)
	}
	return dir
}

// packageOCILayout is a helper that packages a source directory into a
// temporary OCI layout and returns the layout path.
func packageOCILayout(t *testing.T, srcDir string, s *spec.TaitoSpec, tag string) string {
	t.Helper()
	layoutDir := filepath.Join(t.TempDir(), s.Name+"-oci")
	if err := archive.CreateOCILayout(srcDir, layoutDir, tag, s); err != nil {
		t.Fatalf("CreateOCILayout(%s): %v", s.Name, err)
	}
	return layoutDir
}

// fakeToolRoot creates a temporary tool root directory and returns the
// ToolConfig and root path.
func fakeToolRoot(t *testing.T, name, displayName string) (config.ToolConfig, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	return config.ToolConfig{Name: name, Path: root}, root
}

// --- Integration Tests ---

// TestIntegrationInstallSingleSkillFromExamples packages the example
// git-commit-helper skill into an OCI layout, installs it to two fake tools,
// and verifies the files were extracted correctly and installed.json is updated.
func TestIntegrationInstallSingleSkillFromExamples(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	srcDir := singleSkillDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "git-commit-helper:latest")
	ref := NormalizeReference("registry.gitlab.com/org/git-commit-helper:latest")

	assertSpecReadback(t, layoutDir, "git-commit-helper", spec.TypeSkill)

	// Set up two fake tools (simulating Cursor and Claude Code).
	tool1, root1 := fakeToolRoot(t, "cursor", "Cursor")
	tool2, root2 := fakeToolRoot(t, "claude-code", "Claude Code")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	s1, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir, ref, s1, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (1 skill x 2 tools), got %d", len(results))
	}

	assertSkillFilesInRoots(t, []string{root1, root2}, "git-commit-helper", ref)
	assertAllResultsMatch(t, results, "git-commit-helper", spec.TypeSkill)
	assertSingleInstalled(t, "git-commit-helper", ref)
}

// TestIntegrationInstallSingleAgentFromExamples packages the example
// devops-agent and installs it, verifying it goes to agents/ (not skills/).
func TestIntegrationInstallSingleAgentFromExamples(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	srcDir := singleAgentDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "devops-agent:1.0.0")

	tool, root := fakeToolRoot(t, "cursor", "Cursor")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s2, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir, NormalizeReference("registry.example.com/devops-agent:1.0.0"), s2, cfg)
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

	// Agent must be in agents/ directory.
	agentDir := mustResolveToolTarget(t, root, spec.TypeAgent, "devops-agent",
		NormalizeReference("registry.example.com/devops-agent:1.0.0"))
	assertFileExists(t, filepath.Join(agentDir, "taito.spec"), 0, "taito.spec")
	assertFileExists(t, filepath.Join(agentDir, "SKILL.md"), 0, "SKILL.md")

	// Must NOT have a skills/ directory — it's an agent.
	assertNoDir(t, filepath.Join(root, "skills"), "skills")
}

// TestIntegrationInstallBundleFromExamples exercises the full bundle flow:
// package the example bundle-repo → install → verify all children are
// correctly extracted to the right directories in multiple tools.
func TestIntegrationInstallBundleFromExamples(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	srcDir := bundleRepoDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "devtools-bundle:latest")

	assertBundleReadback(t, layoutDir, "devtools-bundle", 3)

	// Set up two fake tools.
	tool1, root1 := fakeToolRoot(t, "cursor", "Cursor")
	tool2, root2 := fakeToolRoot(t, "claude-code", "Claude Code")
	cfg := &config.Config{Tools: []config.ToolConfig{tool1, tool2}}

	ref := NormalizeReference("registry.gitlab.com/org/devtools-bundle:latest")
	s3, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir, ref, s3, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Bundle has 3 children × 2 tools = 6 results.
	if len(results) != 6 {
		t.Fatalf("expected 6 results (3 children × 2 tools), got %d", len(results))
	}

	assertResultTypeCounts(t, results, 4, 2)
	assertBundleFilesInRoots(t, []string{root1, root2}, ref)
	assertInstalledEntries(t,
		[]string{"doc-generator", "git-commit-helper", "devops-agent"},
		[]string{"cursor", "claude-code"},
		ref,
	)
}

// TestIntegrationInstallBundleReinstallIsClean verifies that reinstalling a
// bundle removes stale files from the previous installation.
func TestIntegrationInstallBundleReinstallIsClean(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	srcDir := bundleRepoDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "devtools-bundle:latest")

	tool, root := fakeToolRoot(t, "cursor", "Cursor")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	// First install.
	s5, _ := ReadSpecFromLayout(layoutDir)
	if _, err := Install(layoutDir, "ref", s5, cfg); err != nil {
		t.Fatalf("Install 1: %v", err)
	}

	// Drop a sentinel file in one of the child directories.
	sentinelDir := mustResolveToolTarget(t, root, spec.TypeSkill, "doc-generator", "ref")
	sentinel := filepath.Join(sentinelDir, "stale.txt")
	if err := os.WriteFile(sentinel, []byte("should be removed"), 0644); err != nil {
		t.Fatal(err)
	}

	// Reinstall.
	s6, _ := ReadSpecFromLayout(layoutDir)
	if _, err := Install(layoutDir, "ref", s6, cfg); err != nil {
		t.Fatalf("Install 2: %v", err)
	}

	// Sentinel must be gone.
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("stale file should have been removed on reinstall")
	}

	// But real files must still be there.
	docGenDir := mustResolveToolTarget(t, root, spec.TypeSkill, "doc-generator", "ref")
	assertFileExists(t, filepath.Join(docGenDir, "taito.spec"), 0, "doc-generator taito.spec after reinstall")
}

// TestIntegrationInstallFromLocalOCILayout verifies that IsOCILayout correctly
// identifies a layout produced by CreateOCILayout, and that the full install
// pipeline works from it.
func TestIntegrationInstallFromLocalOCILayout(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	srcDir := singleSkillDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "git-commit-helper:latest")

	// The layout should be detected as an OCI layout.
	if !IsOCILayout(layoutDir) {
		t.Fatalf("expected IsOCILayout(%s) to return true", layoutDir)
	}

	// Install from the local layout (empty reference for local installs).
	tool, root := fakeToolRoot(t, "opencode", "OpenCode")
	cfg := &config.Config{Tools: []config.ToolConfig{tool}}

	s4, _ := ReadSpecFromLayout(layoutDir)
	results, err := Install(layoutDir, "", s4, cfg)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	skillDir := mustResolveToolTarget(t, root, spec.TypeSkill, "git-commit-helper", "")
	assertFileExists(t, filepath.Join(skillDir, "SKILL.md"), 0, "SKILL.md")

	// Verify installed entry has empty reference for local installs.
	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	if (len(idx.Installed.Skills) + len(idx.Installed.Agents)) != 1 {
		t.Fatalf("expected 1 installed entry, got %d", (len(idx.Installed.Skills) + len(idx.Installed.Agents)))
	}
	if idx.Installed.Skills[0].Reference != "" {
		t.Errorf("expected empty reference for local install, got %q", idx.Installed.Skills[0].Reference)
	}
}

// TestIntegrationInstallNoToolsError verifies the error message when no tools
// are configured.
func TestIntegrationInstallNoToolsError(t *testing.T) {
	srcDir := singleSkillDir(t)
	s, err := spec.Load(filepath.Join(srcDir, "taito.spec"))
	if err != nil {
		t.Fatalf("spec.Load: %v", err)
	}

	layoutDir := packageOCILayout(t, srcDir, s, "test:latest")
	cfg := &config.Config{Tools: []config.ToolConfig{}}

	s7, _ := ReadSpecFromLayout(layoutDir)
	_, err = Install(layoutDir, "ref", s7, cfg)
	if err == nil {
		t.Fatal("expected error when no tools configured")
	}
	expected := "no tools configured — run 'taito setup' first"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

// --- Helpers ---

func assertFileExists(t *testing.T, path string, toolIndex int, desc string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("tool %d: expected %s at %s: %v", toolIndex, desc, path, err)
	}
}

// mustResolveToolTarget resolves the install path for a spec type/name in a
// tool root. It normalises the reference and fatals on error.
func mustResolveToolTarget(t *testing.T, root, specType, name, ref string) string {
	t.Helper()
	p, err := ResolveToolTarget(config.ToolConfig{Name: "fake", Path: root}, specType, name, NormalizeReference(ref))
	if err != nil {
		t.Fatalf("ResolveToolTarget(%s, %s, %s): %v", specType, name, ref, err)
	}
	return p
}

// countResultTypes tallies the number of skill and agent results.
func countResultTypes(results []InstallResult) (skills, agents int) {
	for _, r := range results {
		switch r.SpecType {
		case spec.TypeSkill:
			skills++
		case spec.TypeAgent:
			agents++
		}
	}
	return
}

// assertInstalledEntries verifies that installed.json contains entries for each
// (name, tool) combination and that every entry has the expected reference.
func assertInstalledEntries(t *testing.T, names, tools []string, wantRef string) {
	t.Helper()

	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}

	type key struct{ name, tool string }
	seen := make(map[key]bool)

	allEntries := append(idx.Installed.Skills, idx.Installed.Agents...)
	for _, e := range allEntries {
		for _, loc := range e.InstallIn {
			seen[key{e.Name, loc.Tool}] = true
		}
		if e.Reference != wantRef {
			t.Errorf("installed entry %q: Reference = %q, want %q", e.Name, e.Reference, wantRef)
		}
	}

	for _, name := range names {
		for _, tool := range tools {
			if !seen[key{name, tool}] {
				t.Errorf("missing installed entry (%s, %s)", name, tool)
			}
		}
	}
}

// assertNoDir verifies that a directory does NOT exist.
func assertNoDir(t *testing.T, dir string, desc string) {
	t.Helper()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("%s directory should not exist: %s", desc, dir)
	}
}

// assertSpecReadback verifies that the spec can be read back from the OCI
// layout and has the expected name and type.
func assertSpecReadback(t *testing.T, layoutDir, wantName, wantType string) {
	t.Helper()
	readBack, err := ReadSpecFromLayout(layoutDir)
	if err != nil {
		t.Fatalf("ReadSpecFromLayout: %v", err)
	}
	if readBack.Name != wantName {
		t.Errorf("Name = %q, want %q", readBack.Name, wantName)
	}
	if readBack.Type != wantType {
		t.Errorf("Type = %q, want %q", readBack.Type, wantType)
	}
}

// assertBundleReadback verifies the spec is a bundle with the expected name and
// includes count.
func assertBundleReadback(t *testing.T, layoutDir, wantName string, wantIncludes int) {
	t.Helper()
	readBack, err := ReadSpecFromLayout(layoutDir)
	if err != nil {
		t.Fatalf("ReadSpecFromLayout: %v", err)
	}
	if readBack.Type != spec.TypeBundle {
		t.Errorf("Type = %q, want %q", readBack.Type, spec.TypeBundle)
	}
	if readBack.Name != wantName {
		t.Errorf("Name = %q, want %q", readBack.Name, wantName)
	}
	if len(readBack.Includes) != wantIncludes {
		t.Errorf("Includes count = %d, want %d", len(readBack.Includes), wantIncludes)
	}
}

// assertSkillFilesInRoots verifies that a skill was extracted correctly into
// each tool root — taito.spec and SKILL.md exist, and agents/ does not.
func assertSkillFilesInRoots(t *testing.T, roots []string, name, ref string) {
	t.Helper()
	for i, root := range roots {
		dir := mustResolveToolTarget(t, root, spec.TypeSkill, name, ref)
		assertFileExists(t, filepath.Join(dir, "taito.spec"), i, name+" taito.spec")
		assertFileExists(t, filepath.Join(dir, "SKILL.md"), i, name+" SKILL.md")
		assertNoDir(t, filepath.Join(root, "agents"), "agents")
	}
}

// assertBundleFilesInRoots verifies that all bundle children (2 skills + 1
// agent) were extracted correctly into each tool root.
func assertBundleFilesInRoots(t *testing.T, roots []string, ref string) {
	t.Helper()
	type child struct {
		specType string
		name     string
		extra    []string // additional files beyond taito.spec + SKILL.md
	}
	children := []child{
		{spec.TypeSkill, "doc-generator", nil},
		{spec.TypeSkill, "git-commit-helper", nil},
		{spec.TypeAgent, "devops-agent", []string{"testfile"}},
	}
	for i, root := range roots {
		for _, c := range children {
			dir := mustResolveToolTarget(t, root, c.specType, c.name, ref)
			assertFileExists(t, filepath.Join(dir, "taito.spec"), i, c.name+" taito.spec")
			assertFileExists(t, filepath.Join(dir, "SKILL.md"), i, c.name+" SKILL.md")
			for _, f := range c.extra {
				assertFileExists(t, filepath.Join(dir, f), i, c.name+" "+f)
			}
		}
	}
}

// assertAllResultsMatch verifies every result has the expected name and type.
func assertAllResultsMatch(t *testing.T, results []InstallResult, wantName, wantType string) {
	t.Helper()
	for _, r := range results {
		if r.Name != wantName {
			t.Errorf("result Name = %q, want %q", r.Name, wantName)
		}
		if r.SpecType != wantType {
			t.Errorf("result SpecType = %q, want %q", r.SpecType, wantType)
		}
	}
}

// assertResultTypeCounts verifies the number of skill and agent results.
func assertResultTypeCounts(t *testing.T, results []InstallResult, wantSkills, wantAgents int) {
	t.Helper()
	skills, agents := countResultTypes(results)
	if skills != wantSkills {
		t.Errorf("expected %d skill results, got %d", wantSkills, skills)
	}
	if agents != wantAgents {
		t.Errorf("expected %d agent results, got %d", wantAgents, agents)
	}
}

// assertSingleInstalled verifies that installed.json has exactly one entry
// with the expected name and reference.
func assertSingleInstalled(t *testing.T, wantName, wantRef string) {
	t.Helper()
	idx, err := LoadInstalled()
	if err != nil {
		t.Fatalf("LoadInstalled: %v", err)
	}
	total := len(idx.Installed.Skills) + len(idx.Installed.Agents)
	if total != 1 {
		t.Fatalf("expected 1 installed entry, got %d", total)
	}
	allEntries := append(idx.Installed.Skills, idx.Installed.Agents...)
	e := allEntries[0]
	if e.Name != wantName {
		t.Errorf("Name = %q, want %q", e.Name, wantName)
	}
	if e.Reference != wantRef {
		t.Errorf("Reference = %q, want %q", e.Reference, wantRef)
	}
	if e.InstalledAt == "" {
		t.Error("InstalledAt should not be empty")
	}
}
