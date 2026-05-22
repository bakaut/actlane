package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateGithubDraftPROpenCodePack(t *testing.T) {
	root := repoRoot(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"validate", filepath.Join(root, "packs/create-github-draft-pr")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("validate output should include valid, got %q", stdout.String())
	}
}

func TestGenerateOpenCodeWritesOnlyGeneratedOutput(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	assertExists(t, filepath.Join(packDir, "generated/opencode/opencode.jsonc"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/AGENT.MD"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/AGENTS.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/SKILLS.MD"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/commands/create-github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/agents/github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	assertExists(t, filepath.Join(packDir, "generated/mcp/tools.json"))
	assertExists(t, filepath.Join(packDir, "generated/mcp/server.json"))
	assertExists(t, filepath.Join(packDir, "generated/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/actlane.lock"))
	assertNotExists(t, filepath.Join(packDir, ".opencode"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))
	assertNotExists(t, filepath.Join(packDir, "generated/AGENT.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/SKILLS.md"))

	snippet := readFile(t, filepath.Join(packDir, "generated/opencode/opencode.jsonc"))
	for _, want := range []string{
		`"$schema": "https://opencode.ai/config.json"`,
		`"edit": "ask"`,
		`"bash": "ask"`,
		`"skill": "allow"`,
		`"mcp":`,
		`"github":`,
		`"type": "local"`,
		`"environment":`,
		`"ghcr.io/github/github-mcp-server"`,
		`"GITHUB_TOOLS": "create_branch,create_pull_request,push_files"`,
	} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("generated snippet missing %q:\n%s", want, snippet)
		}
	}

	skills := readFile(t, filepath.Join(packDir, "generated/opencode/SKILLS.MD"))
	for _, want := range []string{
		"create-github-draft-pr",
		"Restrictions",
		"Declared MCP tools",
		"github.create_pull_request",
	} {
		if !strings.Contains(skills, want) {
			t.Fatalf("generated SKILLS.md missing %q:\n%s", want, skills)
		}
	}

	agent := readFile(t, filepath.Join(packDir, "generated/opencode/AGENT.MD"))
	for _, want := range []string{
		"<!-- actlane:generated capability=create-github-draft-pr target=opencode -->",
		"Inspect the current project",
		"Refuse to modify existing project-owned files",
		"<!-- actlane:start create-github-draft-pr.<block-id> -->",
	} {
		if !strings.Contains(agent, want) {
			t.Fatalf("generated AGENT.MD missing %q:\n%s", want, agent)
		}
	}

	mcpTools := readFile(t, filepath.Join(packDir, "generated/mcp/tools.json"))
	for _, want := range []string{
		`"serverSource": "https://github.com/github/github-mcp-server"`,
		`"name": "create_branch"`,
		`"name": "push_files"`,
		`"name": "create_pull_request"`,
	} {
		if !strings.Contains(mcpTools, want) {
			t.Fatalf("generated MCP tools missing %q:\n%s", want, mcpTools)
		}
	}

	mcpServer := readFile(t, filepath.Join(packDir, "generated/mcp/server.json"))
	for _, want := range []string{
		`"command": [`,
		`"ghcr.io/github/github-mcp-server"`,
		`"GITHUB_PERSONAL_ACCESS_TOKEN"`,
		`"{env:GITHUB_PERSONAL_ACCESS_TOKEN}"`,
		`"GITHUB_TOOLS": "create_branch,create_pull_request,push_files"`,
		`"type": "local"`,
		`"environment":`,
	} {
		if !strings.Contains(mcpServer, want) {
			t.Fatalf("generated MCP server config missing %q:\n%s", want, mcpServer)
		}
	}
}

func TestGenerateCheckDetectsStaleOutputWithoutWriting(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode", "--check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--check should fail when generated output is missing")
	}
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode", "--check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("--check should pass after generation\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
}

func TestFrozenLockfileDetectsDrift(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	capabilityPath := filepath.Join(packDir, "capabilities/create-github-draft-pr.yaml")
	content := readFile(t, capabilityPath)
	content = strings.Replace(content, "Safely prepare a GitHub draft pull request", "Prepare a GitHub draft pull request", 1)
	if err := os.WriteFile(capabilityPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode", "--frozen-lockfile"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--frozen-lockfile should fail after source drift")
	}
	if !strings.Contains(stderr.String(), "lockfile") {
		t.Fatalf("expected lockfile drift error, got %q", stderr.String())
	}
}

func TestFrozenLockfileDetectsProfileSourceDrift(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	sourcePath := filepath.Join(packDir, "capabilities/AGENTS.md")
	content := readFile(t, sourcePath)
	if err := os.WriteFile(sourcePath, []byte(content+"\nDrift marker.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "opencode", "--frozen-lockfile"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--frozen-lockfile should fail after profile source drift")
	}
	if !strings.Contains(stderr.String(), "lockfile") {
		t.Fatalf("expected lockfile drift error, got %q", stderr.String())
	}
}

func TestUnsupportedTargetFailsClearly(t *testing.T) {
	root := repoRoot(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", filepath.Join(root, "packs/create-github-draft-pr"), "--target", "mcp"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("unsupported target should fail")
	}
	if !strings.Contains(stderr.String(), "supported targets: opencode") {
		t.Fatalf("unsupported target error should list opencode, got %q", stderr.String())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "packs/create-github-draft-pr")); err == nil {
			return wd
		}
		next := filepath.Dir(wd)
		if next == wd {
			t.Fatal("repository root not found")
		}
		wd = next
	}
}

func copyPackToTemp(t *testing.T) string {
	t.Helper()
	src := filepath.Join(repoRoot(t), "packs/create-github-draft-pr")
	dst := filepath.Join(t.TempDir(), "create-github-draft-pr")
	copyDir(t, src, dst)
	if err := os.RemoveAll(filepath.Join(dst, "generated")); err != nil {
		t.Fatal(err)
	}
	return dst
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dstPath, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected stat error for %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
