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
	assertExists(t, filepath.Join(packDir, "generated/opencode/AGENTS.md"))
	assertExists(t, filepath.Join(packDir, "generated/mcp/tools.json"))
	assertExists(t, filepath.Join(packDir, "generated/mcp/server.json"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/actlane.lock"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/commands/create-github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/agents/github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/actlane.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/capabilities"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/policies/github-draft-pr.policy.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/mcp"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/target-profiles"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/files"))
	assertNotExists(t, filepath.Join(packDir, ".opencode"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/opencode.snippet.jsonc"))
	assertNotExists(t, filepath.Join(packDir, "generated/AGENT.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/AGENT.MD"))
	assertNotExists(t, filepath.Join(packDir, "generated/SKILLS.md"))

	snippet := readFile(t, filepath.Join(packDir, "generated/opencode/opencode.jsonc"))
	for _, want := range []string{
		`"$schema": "https://opencode.ai/config.json"`,
		`"edit": "ask"`,
		`"bash": "ask"`,
		`"skill": "allow"`,
		`"mcp":`,
		`"actlane-safe-gitops":`,
		`"github":`,
		`"type": "local"`,
		`"enabled": true`,
		`"actlane"`,
		`"--policy-bundle"`,
		`"./policies/policy-bundle.json"`,
		`"docker"`,
		`"ghcr.io/github/github-mcp-server"`,
		`"GITHUB_PERSONAL_ACCESS_TOKEN": "{env:GITHUB_PERSONAL_ACCESS_TOKEN}"`,
		`"GITHUB_TOOLS": "create_branch,push_files,create_pull_request"`,
	} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("generated snippet missing %q:\n%s", want, snippet)
		}
	}

	instructions := readFile(t, filepath.Join(packDir, "generated/opencode/AGENTS.md"))
	for _, want := range []string{
		"Base Agent Instructions",
		"Actlane Instructions",
		"System prompt",
		"Create draft pull requests only",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("generated AGENTS.md missing %q:\n%s", want, instructions)
		}
	}

	command := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/commands/create-github-draft-pr.md"))
	for _, want := range []string{
		`agent: "github-draft-pr"`,
		"Run capability `create-github-draft-pr`",
		"`github_create_pull_request`",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("generated OpenCode command missing %q:\n%s", want, command)
		}
	}

	agent := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/agents/github-draft-pr.md"))
	for _, want := range []string{
		"mode: primary",
		"permission:",
		"actlane:inspect:create-github-draft-pr",
	} {
		if !strings.Contains(agent, want) {
			t.Fatalf("generated OpenCode agent missing %q:\n%s", want, agent)
		}
	}

	skill := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	for _, want := range []string{
		"name: \"create-github-draft-pr\"",
		"compatibility: opencode",
		"execution_ref: \"github-mcp-draft-pr\"",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("generated OpenCode skill missing %q:\n%s", want, skill)
		}
	}
	lockfile := readFile(t, filepath.Join(packDir, "generated/opencode/actlane.lock"))
	if !strings.Contains(lockfile, "skills/create-github-draft-pr.yaml") {
		t.Fatalf("generated lockfile should include skill contract source digest:\n%s", lockfile)
	}

	mcpTools := readFile(t, filepath.Join(packDir, "generated/mcp/tools.json"))
	for _, want := range []string{
		`"binding": "github-mcp-draft-pr"`,
		`"binding": "actlane-safe-gitops"`,
		`"generatedTool": "create_github_draft_pr_audit"`,
		`"generatedTool": "create_github_draft_pr_enforce"`,
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
		`"fromEnv": "GITHUB_PERSONAL_ACCESS_TOKEN"`,
		`"GITHUB_TOOLS": "create_branch,push_files,create_pull_request"`,
		`"transport": "stdio"`,
		`"environment":`,
	} {
		if !strings.Contains(mcpServer, want) {
			t.Fatalf("generated MCP server config missing %q:\n%s", want, mcpServer)
		}
	}
}

func TestGenerateCodexWritesCodexProfile(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate codex failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	assertExists(t, filepath.Join(packDir, "generated/codex/AGENTS.md"))
	assertExists(t, filepath.Join(packDir, "generated/codex/codex.config.toml"))
	assertExists(t, filepath.Join(packDir, "generated/codex/.codex/skills/create-github-draft-pr/SKILL.md"))
	assertExists(t, filepath.Join(packDir, "generated/codex/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/codex/actlane.lock"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/actlane.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/capabilities"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/policies/github-draft-pr.policy.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/mcp"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/target-profiles"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/files"))
	assertNotExists(t, filepath.Join(packDir, ".codex"))

	instructions := readFile(t, filepath.Join(packDir, "generated/codex/AGENTS.md"))
	for _, want := range []string{
		"Base Agent Instructions",
		"Actlane Instructions",
		"System prompt",
	} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("generated Codex AGENTS.md missing %q:\n%s", want, instructions)
		}
	}

	config := readFile(t, filepath.Join(packDir, "generated/codex/codex.config.toml"))
	for _, want := range []string{
		"[mcp_servers.actlane-safe-gitops]",
		`command = "actlane"`,
		`args = ["mcp", "serve", "--policy-bundle", "./policies/policy-bundle.json"]`,
		"[mcp_servers.github]",
		`command = "docker"`,
		`"ghcr.io/github/github-mcp-server"`,
		"[mcp_servers.github.env]",
		`GITHUB_PERSONAL_ACCESS_TOKEN = "${GITHUB_PERSONAL_ACCESS_TOKEN}"`,
		`GITHUB_TOOLS = "create_branch,push_files,create_pull_request"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("generated Codex config missing %q:\n%s", want, config)
		}
	}

	skill := readFile(t, filepath.Join(packDir, "generated/codex/.codex/skills/create-github-draft-pr/SKILL.md"))
	for _, want := range []string{
		"name: \"create-github-draft-pr\"",
		"compatibility: codex",
		"Security gate flow:",
		"`create_github_draft_pr_enforce`",
		"`github_create_pull_request`",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("generated Codex skill missing %q:\n%s", want, skill)
		}
	}
}

func TestMCPServeListsAndCallsPolicyTools(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"create_github_draft_pr_audit","arguments":{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["README.md"],"confirmed":true}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"create_github_draft_pr_enforce","arguments":{"repo":"unknown/repo","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":[".env"],"confirmed":false}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp serve failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"create_github_draft_pr_audit"`,
		`"name":"create_github_draft_pr_enforce"`,
		`\"policyDecision\": \"allow\"`,
		`\"branch\": \"gpt/feature\"`,
		`\"next\":`,
		`\"tool\": \"github_create_branch\"`,
		`\"tool\": \"github_push_files\"`,
		`\"tool\": \"github_create_pull_request\"`,
		`\"policyDecision\": \"deny\"`,
		`"isError":true`,
		`file is forbidden: .env`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp output missing %q:\n%s", want, output)
		}
	}
}

func TestMCPServeAcceptsPolicyBundle(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_github_draft_pr_enforce","arguments":{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["README.md"],"confirmed":true}}}`,
		"",
	}, "\n"))
	stdout.Reset()
	stderr.Reset()
	code = MainWithIO([]string{"mcp", "serve", "--policy-bundle", filepath.Join(packDir, "generated/codex/policies/policy-bundle.json")}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp serve with policy bundle failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"create_github_draft_pr_enforce"`,
		`\"policyDecision\": \"allow\"`,
		`\"branch\": \"gpt/feature\"`,
		`\"next\":`,
		`\"tool\": \"github_create_pull_request\"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp serve with policy bundle missing %q:\n%s", want, output)
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

	sourcePath := filepath.Join(packDir, "files/prompts/AGENTS.md")
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
	if !strings.Contains(stderr.String(), "supported targets: codex, opencode") {
		t.Fatalf("unsupported target error should list codex and opencode, got %q", stderr.String())
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
