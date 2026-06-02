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

func TestValidateContractBoundaries(t *testing.T) {
	cases := []struct {
		name string
		path string
		edit func(string) string
		want string
	}{
		{
			name: "capability-target-profile",
			path: "capabilities/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  profiles:\n    opencode:\n      config: {}\n      files:\n        - path: .opencode/skills/create-github-draft-pr/SKILL.md\n          skillContract: create-github-draft-pr\n"
			},
			want: "must not define spec.profiles",
		},
		{
			name: "skill-generated-mcp-section",
			path: "skills/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return strings.Replace(content, "    Workflow:", "    MCP tools:\n\n    - `github_create_pull_request`\n\n    Workflow:", 1)
			},
			want: "must not embed generated input or MCP tool sections",
		},
		{
			name: "command-safety",
			path: "commands/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  safety:\n    requirePolicy: true\n"
			},
			want: "must not define spec.safety",
		},
		{
			name: "agent-permissions",
			path: "subagents/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  permissions:\n    bash: ask\n"
			},
			want: "must not define spec.permissions",
		},
		{
			name: "responsibility-acceptance-criteria",
			path: "contracts/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  acceptanceCriteria:\n    - keep out of runtime contract\n"
			},
			want: "must not define spec.acceptanceCriteria",
		},
		{
			name: "runtime-target-path",
			path: "runtime-profiles/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  targetPath: .codex/config.toml\n"
			},
			want: "must not define target profile paths",
		},
		{
			name: "evidence-policy-rule",
			path: "evidence/create-github-draft-pr.yaml",
			edit: func(content string) string {
				return content + "\n  deny:\n    - reason: keep policy out\n"
			},
			want: "must not define policy rules",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packDir := copyPackToTemp(t)
			target := filepath.Join(packDir, tc.path)
			content := readFile(t, target)
			if err := os.WriteFile(target, []byte(tc.edit(content)), 0o644); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := Main([]string{"validate", packDir}, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("validate should fail for %s", tc.name)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("validate error missing %q\nstdout:\n%s\nstderr:\n%s", tc.want, stdout.String(), stderr.String())
			}
		})
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
		`description: "Prepare a safe GitHub draft pull request from reviewed changes."`,
		"Use Actlane capability `create-github-draft-pr`.",
		"$ARGUMENTS",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("generated OpenCode command missing %q:\n%s", want, command)
		}
	}

	agent := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/agents/github-draft-pr.md"))
	for _, want := range []string{
		"mode: subagent",
		"Prepare GitHub draft pull requests through Actlane capabilities.",
		"Use capability `create-github-draft-pr`.",
		"Use skill `create-github-draft-pr`.",
		"Raw MCP tools default: `deny`.",
	} {
		if !strings.Contains(agent, want) {
			t.Fatalf("generated OpenCode agent missing %q:\n%s", want, agent)
		}
	}

	skill := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	for _, want := range []string{
		"name: \"create-github-draft-pr\"",
		"description: \"Safely prepare a GitHub draft pull request from reviewed changes.\"",
		"Policy gate tools:",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("generated OpenCode skill missing %q:\n%s", want, skill)
		}
	}
	for _, forbidden := range []string{"compatibility:", "execution_ref:"} {
		if strings.Contains(skill, forbidden) {
			t.Fatalf("generated OpenCode skill should not include %q:\n%s", forbidden, skill)
		}
	}
	lockfile := readFile(t, filepath.Join(packDir, "generated/opencode/actlane.lock"))
	if !strings.Contains(lockfile, "skills/create-github-draft-pr.yaml") {
		t.Fatalf("generated lockfile should include skill contract source digest:\n%s", lockfile)
	}
	if !strings.Contains(lockfile, "commands/create-github-draft-pr.yaml") {
		t.Fatalf("generated lockfile should include command contract source digest:\n%s", lockfile)
	}
	if !strings.Contains(lockfile, "subagents/create-github-draft-pr.yaml") {
		t.Fatalf("generated lockfile should include agent contract source digest:\n%s", lockfile)
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
		"[mcp_servers.actlane-pack-author]",
		`args = ["mcp", "author", "serve", "--pack", "./packs/create-github-draft-pr"]`,
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
		"description: \"Safely prepare a GitHub draft pull request from reviewed changes.\"",
		"Policy gate tools:",
		"`create_github_draft_pr_enforce`",
		"`github_create_pull_request`",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("generated Codex skill missing %q:\n%s", want, skill)
		}
	}
}

func TestGenerateSkillContractResources(t *testing.T) {
	packDir := copyPackToTemp(t)
	referenceDir := filepath.Join(packDir, "skills/references")
	if err := os.MkdirAll(referenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	referencePath := filepath.Join(referenceDir, "usage.md")
	if err := os.WriteFile(referencePath, []byte("# Usage\n\nResource content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillPath := filepath.Join(packDir, "skills/create-github-draft-pr.yaml")
	skill := readFile(t, skillPath)
	skill += "\n  references:\n    - source: references/usage.md\n      path: references/usage.md\n"
	if err := os.WriteFile(skillPath, []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	generatedReference := filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/references/usage.md")
	assertExists(t, generatedReference)
	if got := readFile(t, generatedReference); !strings.Contains(got, "Resource content.") {
		t.Fatalf("generated reference missing content:\n%s", got)
	}
	lockfile := readFile(t, filepath.Join(packDir, "generated/opencode/actlane.lock"))
	if !strings.Contains(lockfile, "skills/references/usage.md") {
		t.Fatalf("generated lockfile should include skill resource source digest:\n%s", lockfile)
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

func TestMCPServeClassifiesWithRuntimeAndEvidenceContracts(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_classify","arguments":{"task":"Prepare a safe GitHub draft PR for reviewed README changes","changed_files":["README.md"],"branch":"main","diff_summary":"docs only update"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_classify","arguments":{"task":"Create PR with .env token update","changed_files":[".env"],"branch":"main","diff_summary":"SECRET_TOKEN changed"}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp classify failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_classify"`,
		`\"workType\": \"docs_change\"`,
		`\"mode\": \"advise\"`,
		`\"candidateCapabilities\": [`,
		`\"create-github-draft-pr\"`,
		`\"requiredEvidence\": [`,
		`\"policy_decision\"`,
		`\"draft_pr_url\"`,
		`\"riskFlags\": [`,
		`\"secrets_sensitive\"`,
		`\"mode\": \"read-only\"`,
		`human boundary`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp classify output missing %q:\n%s", want, output)
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

func TestMCPAuthorServeExposesPackAuthoringTools(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_pack_inspect","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"actlane_pack_validate","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"actlane_pack_generate_preview","arguments":{"target":"codex"}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"actlane_pack_plan_change","arguments":{"name":"safe-deploy","targets":["codex","opencode"]}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"actlane_pack_apply_change","arguments":{"confirmed":false,"files":[{"path":"skills/safe-apply.yaml","content":"apiVersion: actlane.ru/v1alpha1\nkind: SkillContract\nmetadata:\n  name: safe-apply\nspec:\n  body: |\n    Test.\n"}]}}}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"actlane_pack_apply_change","arguments":{"confirmed":true,"files":[{"path":"skills/safe-apply.yaml","content":"apiVersion: actlane.ru/v1alpha1\nkind: SkillContract\nmetadata:\n  name: safe-apply\nspec:\n  body: |\n    Test.\n"}]}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "author", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp author serve failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane-pack-author"`,
		`"name":"actlane_pack_inspect"`,
		`"name":"actlane_pack_validate"`,
		`"name":"actlane_pack_plan_change"`,
		`"name":"actlane_pack_apply_change"`,
		`\"name\": \"github-draft-pr-pack\"`,
		`\"valid\": true`,
		`\"target\": \"codex\"`,
		`\"path\": \"generated/codex/AGENTS.md\"`,
		`\"path\": \"capabilities/safe-deploy.yaml\"`,
		`\"path\": \"target-profiles/opencode.yaml\"`,
		`\"mutationPermitted\": false`,
		`confirmed must be true`,
		`\"written\": [`,
		`\"skills/safe-apply.yaml\"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp author output missing %q:\n%s", want, output)
		}
	}
	if got := readFile(t, filepath.Join(packDir, "skills/safe-apply.yaml")); !strings.Contains(got, "name: safe-apply") {
		t.Fatalf("mcp author apply did not write expected source file:\n%s", got)
	}
}

func TestPlanCodexSafeAdoptionDetectsCreatesAndConflicts(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "# Existing guidance\n")
	writeTestFile(t, filepath.Join(projectDir, ".codex/skills/create-github-draft-pr/SKILL.md"), "user-owned skill\n")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{
		"plan",
		packDir,
		"--target", "codex",
		"--from", filepath.Join(packDir, "generated/codex"),
		"--project", projectDir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("plan failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Will append Actlane block:",
		"AGENTS.md",
		"Will create:",
		".codex/config.toml",
		"policies/policy-bundle.json",
		".codex/skills/dushnila/SKILL.md",
		"source:",
		"preview:",
		"sha256:",
		"Conflicts:",
		".codex/skills/create-github-draft-pr/SKILL.md",
		"Apply blocked: 1 conflict(s)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan output missing %q:\n%s", want, output)
		}
	}
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/dushnila/SKILL.md"))
}

func TestPlanRequiresTargetAndDefaultsGeneratedAndCurrentProject(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"plan", packDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("plan without target should fail\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "--target is required") {
		t.Fatalf("plan without target error missing --target requirement:\n%s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"plan", packDir, "--target", "codex"}, &stdout, &stderr)
	if chdirErr := os.Chdir(wd); chdirErr != nil {
		t.Fatal(chdirErr)
	}
	if code != 0 {
		t.Fatalf("plan with defaults failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Plan target: codex",
		"Generated: " + filepath.Join(packDir, "generated/codex"),
		".codex/skills/create-github-draft-pr/SKILL.md",
		".codex/skills/dushnila/SKILL.md",
		".codex/config.toml",
		"policies/policy-bundle.json",
		"AGENTS.md",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("default plan output missing %q:\n%s", want, output)
		}
	}
}

func TestApplyCodexSafeAdoptionCreatesAndUpdatesIdempotently(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "# Existing guidance\n")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		".codex/skills/create-github-draft-pr/SKILL.md",
		".codex/skills/dushnila/SKILL.md",
		".codex/config.toml",
		"policies/policy-bundle.json",
	} {
		assertExists(t, filepath.Join(projectDir, path))
	}
	config := readFile(t, filepath.Join(projectDir, ".codex/config.toml"))
	for _, want := range []string{
		"# actlane:start github-draft-pr-pack/.codex/config.toml",
		`args = ["mcp", "serve", "--policy-bundle", "./policies/policy-bundle.json"]`,
		"# actlane:end github-draft-pr-pack/.codex/config.toml",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("codex.config.toml missing %q:\n%s", want, config)
		}
	}
	policy := readFile(t, filepath.Join(projectDir, "policies/policy-bundle.json"))
	if !strings.Contains(policy, `"mcpBindings"`) || !strings.Contains(policy, `"actlane-safe-gitops"`) {
		t.Fatalf("policy bundle should include MCP bindings:\n%s", policy)
	}
	agents := readFile(t, filepath.Join(projectDir, "AGENTS.md"))
	for _, want := range []string{
		"# Existing guidance",
		"<!-- actlane:start github-draft-pr-pack/AGENTS.md -->",
		"<!-- actlane:end github-draft-pr-pack/AGENTS.md -->",
	} {
		if !strings.Contains(agents, want) {
			t.Fatalf("AGENTS.md missing %q:\n%s", want, agents)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	agents = readFile(t, filepath.Join(projectDir, "AGENTS.md"))
	if strings.Count(agents, "<!-- actlane:start github-draft-pr-pack/AGENTS.md -->") != 1 {
		t.Fatalf("second apply duplicated AGENTS marker:\n%s", agents)
	}
	config = readFile(t, filepath.Join(projectDir, ".codex/config.toml"))
	if strings.Count(config, "# actlane:start github-draft-pr-pack/.codex/config.toml") != 1 {
		t.Fatalf("second apply duplicated Codex config marker:\n%s", config)
	}
	if !strings.Contains(stdout.String(), "Skipped:") {
		t.Fatalf("second apply should skip unchanged owned output:\n%s", stdout.String())
	}
}

func TestApplyCodexDryRunWritesNothingAndConflictsBlock(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir, "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply dry-run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/dushnila/SKILL.md"))
	assertNotExists(t, filepath.Join(projectDir, ".codex/config.toml"))
	assertNotExists(t, filepath.Join(projectDir, "policies/policy-bundle.json"))

	writeTestFile(t, filepath.Join(projectDir, ".codex/skills/dushnila/SKILL.md"), "user-owned\n")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("apply should fail on conflict\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conflicts:") || !strings.Contains(stderr.String(), "apply blocked") {
		t.Fatalf("apply conflict output missing details\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/create-github-draft-pr/SKILL.md"))
}

func TestRemoveCodexSafeAdoptionRemovesOnlyOwnedArtifacts(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "# Existing guidance\n")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "codex", "--project", projectDir, "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remove dry-run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(projectDir, ".codex/skills/dushnila/SKILL.md"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remove failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(projectDir, ".codex/skills/dushnila/SKILL.md"))
	assertNotExists(t, filepath.Join(projectDir, ".codex/config.toml"))
	assertNotExists(t, filepath.Join(projectDir, "policies/policy-bundle.json"))
	agents := readFile(t, filepath.Join(projectDir, "AGENTS.md"))
	if !strings.Contains(agents, "# Existing guidance") {
		t.Fatalf("remove should preserve user AGENTS content:\n%s", agents)
	}
	if strings.Contains(agents, "actlane:start") {
		t.Fatalf("remove should delete Actlane AGENTS block:\n%s", agents)
	}
}

func TestRemoveCodexBlocksUserModifiedGeneratedFile(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	writeTestFile(t, filepath.Join(projectDir, "policies/policy-bundle.json"), "user modified\n")

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("remove should fail on modified generated file\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conflicts:") || !strings.Contains(stderr.String(), "remove blocked") {
		t.Fatalf("remove conflict output missing details\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(projectDir, ".codex/skills/create-github-draft-pr/SKILL.md"))
	assertExists(t, filepath.Join(projectDir, "policies/policy-bundle.json"))
}

func TestPlanCodexSafeAdoptionJSON(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{
		"plan",
		packDir,
		"--target", "codex",
		"--from", filepath.Join(packDir, "generated/codex"),
		"--project", projectDir,
		"--json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("plan json failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"target": "codex"`,
		`"action": "create_file"`,
		`"targetPath": ".codex/skills/dushnila/SKILL.md"`,
		`"targetPath": ".codex/config.toml"`,
		`"markerStyle": "hash"`,
		`"targetPath": "policies/policy-bundle.json"`,
		`"preview":`,
		`"sha256":`,
		`"conflicts": 0`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan json missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{
		`"diff":`,
		`"content":`,
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("plan json should not include %q without explicit flag:\n%s", unwanted, output)
		}
	}
}

func TestPlanCodexSafeAdoptionDiffAndContent(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{
		"plan",
		packDir,
		"--target", "codex",
		"--from", filepath.Join(packDir, "generated/codex"),
		"--project", projectDir,
		"--diff",
		"--show-content",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("plan diff failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"diff:",
		"--- /dev/null",
		"+++ .codex/skills/dushnila/SKILL.md",
		"+++ .codex/config.toml",
		"content:",
		"name: \"dushnila\"",
		"# actlane:start github-draft-pr-pack/.codex/config.toml",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan diff output missing %q:\n%s", want, output)
		}
	}
}

func TestCheckUsesResponsibilityContract(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(`{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":[".github/workflows/release.yml"],"confirmed":true}`)
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"check", "--pack", packDir, "--capability", "create-github-draft-pr"}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"policyDecision": "requires_approval"`,
		`"risk": "high"`,
		`"ci"`,
		`"security-scan"`,
		`"humanApprovalRequired": true`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("check output missing %q:\n%s", want, output)
		}
	}
}

func TestCheckEnforceStopsCriticalRisk(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(`{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["secrets/token.txt"],"confirmed":true}`)
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"check", "--pack", packDir, "--capability", "create-github-draft-pr", "--mode", "enforce"}, stdin, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("check enforce should fail with code 1, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"policyDecision": "deny"`,
		`"risk": "critical"`,
		`"stop": true`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("check enforce output missing %q:\n%s", want, output)
		}
	}
}

func TestGeneratedPolicyBundleCarriesResponsibilityContract(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	bundle := readFile(t, filepath.Join(packDir, "generated/codex/policies/policy-bundle.json"))
	for _, want := range []string{
		`"responsibility":`,
		`"riskFloor": "critical"`,
		`"requiredForHandoff"`,
	} {
		if !strings.Contains(bundle, want) {
			t.Fatalf("policy bundle missing %q:\n%s", want, bundle)
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

func TestInspectImportPackInstallAndGenerateDefaultTarget(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "Project agent guidance.\n")
	writeTestFile(t, filepath.Join(projectDir, "opencode.jsonc"), `{
  "$schema": "https://opencode.ai/config.json",
  "permission": {
    "bash": "ask",
    "edit": "ask",
    "skill": "allow",
    "github_create_pull_request": "allow"
  },
  "mcp": {
    "github": {
      "type": "local",
      "command": ["github-mcp-server"]
    }
  }
}
`)
	writeTestFile(t, filepath.Join(projectDir, ".opencode/commands/create-github-draft-pr.md"), `---
agent: github-draft-pr
description: Prepare a GitHub draft pull request.
---

Create a draft PR from $ARGUMENTS.
`)
	writeTestFile(t, filepath.Join(projectDir, ".opencode/agents/github-draft-pr.md"), `---
description: GitHub draft PR specialist.
---

Prepare safe draft PRs.
`)
	writeTestFile(t, filepath.Join(projectDir, ".opencode/skills/create-github-draft-pr/SKILL.md"), `---
name: create-github-draft-pr
description: Draft PR skill.
---

Use the GitHub draft PR workflow.
`)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"inspect", "--from", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"ai-agent: opencode", "command: create-github-draft-pr", "agent: github-draft-pr", "skill: create-github-draft-pr", "mcp server: github"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("inspect output missing %q:\n%s", want, stdout.String())
		}
	}

	actlaneDir := filepath.Join(t.TempDir(), ".actlane")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"import", "--from", projectDir, "--out", actlaneDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(actlaneDir, "actlane.yaml"))
	assertExists(t, filepath.Join(actlaneDir, "capabilities/create-github-draft-pr.yaml"))
	assertExists(t, filepath.Join(actlaneDir, "commands/create-github-draft-pr.yaml"))
	assertExists(t, filepath.Join(actlaneDir, "agents/github-draft-pr.yaml"))
	assertExists(t, filepath.Join(actlaneDir, "skills/create-github-draft-pr.yaml"))
	assertExists(t, filepath.Join(actlaneDir, "import.report.md"))
	mcpBinding := readFile(t, filepath.Join(actlaneDir, "mcp/bindings/create-github-draft-pr.yaml"))
	if !strings.Contains(mcpBinding, "github-mcp-server") {
		t.Fatalf("imported MCP binding should preserve command, got:\n%s", mcpBinding)
	}
	for _, want := range []string{"requiredTools:", "name: github_create_pull_request", "server: github", "toolset: opencode-permission"} {
		if !strings.Contains(mcpBinding, want) {
			t.Fatalf("imported MCP binding should preserve MCP tools %q, got:\n%s", want, mcpBinding)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"validate", actlaneDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate imported pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"import", "report", "--from", actlaneDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import report failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Detected runtime: opencode") || !strings.Contains(stdout.String(), "Capability, policy, and MCP binding were inferred") {
		t.Fatalf("unexpected import report:\n%s", stdout.String())
	}

	archive := filepath.Join(t.TempDir(), "actlane-pack.zip")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"pack", "create", "--from", actlaneDir, "--out", archive}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack create failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, archive)

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"pack", "inspect", archive}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack inspect failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Source runtime: opencode") || !strings.Contains(stdout.String(), "Capability") || !strings.Contains(stdout.String(), "codex") {
		t.Fatalf("unexpected pack inspect output:\n%s", stdout.String())
	}

	consumerDir := filepath.Join(t.TempDir(), "consumer")
	if err := os.MkdirAll(consumerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	archiveData, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumerDir, "actlane-pack.zip"), archiveData, 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(consumerDir); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", "--target", "codex"}, &stdout, &stderr)
	if chdirErr := os.Chdir(wd); chdirErr != nil {
		t.Fatal(chdirErr)
	}
	if code != 0 {
		t.Fatalf("generate from default pack zip failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(consumerDir, "generated/codex/.codex/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(consumerDir, ".actlane"))

	installedDir := filepath.Join(t.TempDir(), ".actlane")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"pack", "install", archive, "--target", "codex", "--out", installedDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack install failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(installedDir, ".local.yaml"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", installedDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate with default target failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(installedDir, "generated/codex/.codex/skills/create-github-draft-pr/SKILL.md"))
	assertExists(t, filepath.Join(installedDir, "generated/codex/actlane.lock"))
}

func TestPackInitCreatesValidScaffoldAndDoesNotOverwrite(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "packs", "safe-deploy")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"pack", "init", "safe-deploy", "--out", outDir, "--targets", "codex,opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack init failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		"actlane.yaml",
		"capabilities/safe-deploy.yaml",
		"policies/safe-deploy-policy.yaml",
		"mcp/bindings/safe-deploy.yaml",
		"skills/safe-deploy.yaml",
		"target-profiles/codex.yaml",
		"target-profiles/opencode.yaml",
	} {
		assertExists(t, filepath.Join(outDir, path))
	}
	for _, path := range []string{
		"commands/safe-deploy.yaml",
		"agents/safe-deploy.yaml",
		"contracts/safe-deploy.yaml",
	} {
		assertNotExists(t, filepath.Join(outDir, path))
	}
	if !strings.Contains(stdout.String(), "initialized pack") || !strings.Contains(stdout.String(), "actlane generate") {
		t.Fatalf("pack init output should include next steps:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"validate", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate initialized pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", outDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate initialized codex pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(outDir, "generated/codex/.codex/skills/safe-deploy/SKILL.md"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"pack", "init", "safe-deploy", "--out", outDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("pack init should not overwrite existing files\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "existing files") {
		t.Fatalf("pack init overwrite error missing details:\n%s", stderr.String())
	}
}

func TestPackInitHelpShowsContracts(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Main([]string{"pack", "init", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack init --help failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "--contracts") || !strings.Contains(stdout.String(), "capability,policy,mcp,skill,target-profile") {
		t.Fatalf("pack init --help should describe contracts option:\n%s", stdout.String())
	}
}

func TestPackInitCreatesRequestedContracts(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "packs", "thefirm")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"pack", "init", "thefirm", "--out", outDir, "--targets", "codex,opencode", "--contracts", "all"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pack init all failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		"actlane.yaml",
		"capabilities/thefirm.yaml",
		"policies/thefirm-policy.yaml",
		"mcp/bindings/thefirm.yaml",
		"skills/thefirm.yaml",
		"commands/thefirm.yaml",
		"agents/thefirm.yaml",
		"contracts/thefirm.yaml",
		"runtime-profiles/thefirm.yaml",
		"evidence/thefirm.yaml",
		"target-profiles/codex.yaml",
		"target-profiles/opencode.yaml",
	} {
		assertExists(t, filepath.Join(outDir, path))
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"validate", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("validate initialized all-contract pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", outDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate initialized opencode all-contract pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(outDir, "generated/opencode/.opencode/commands/thefirm.md"))
	assertExists(t, filepath.Join(outDir, "generated/opencode/.opencode/agents/thefirm.md"))
	assertExists(t, filepath.Join(outDir, "generated/opencode/.opencode/skills/thefirm/SKILL.md"))
}

func TestPackInitRejectsInvalidContractList(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "packs", "broken")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"pack", "init", "broken", "--out", outDir, "--contracts", "capability,policy"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("pack init should reject missing target-profile\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "target-profile") {
		t.Fatalf("pack init missing target-profile error should be explicit:\n%s", stderr.String())
	}
	assertNotExists(t, filepath.Join(outDir, "actlane.yaml"))
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

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
