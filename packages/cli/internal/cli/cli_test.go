package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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

func TestGithubDraftPRPackHasMinimalSourceLayout(t *testing.T) {
	packDir := filepath.Join(repoRoot(t), "packs/create-github-draft-pr")
	for _, path := range []string{
		"actlane.yaml",
		"capabilities/create-github-draft-pr.yaml",
		"commands/create-github-draft-pr.yaml",
		"evidence/create-github-draft-pr.yaml",
		"mcp/bindings/actlane-mcp-broker.yaml",
		"mcp/bindings/github-mcp-draft-pr.yaml",
		"policies/github-draft-pr.policy.yaml",
		"skills/create-github-draft-pr.yaml",
		"target-profiles/codex.yaml",
		"target-profiles/opencode.yaml",
	} {
		assertExists(t, filepath.Join(packDir, path))
	}
	for _, path := range []string{
		"agents/create-github-draft-pr.yaml",
		"contracts/create-github-draft-pr.yaml",
		"runtime-profiles/create-github-draft-pr.yaml",
		"subagents/create-github-draft-pr.yaml",
		"mcp/bindings/actlane-safe-gitops.yaml",
		"mcp/bindings/actlane-pack-author.yaml",
		"skills/dushnila.yaml",
		"skills/scope-governor.yaml",
	} {
		assertNotExists(t, filepath.Join(packDir, path))
	}
	githubBinding := readFile(t, filepath.Join(packDir, "mcp/bindings/github-mcp-draft-pr.yaml"))
	if !strings.Contains(githubBinding, "exposeToAgent: false") {
		t.Fatalf("downstream GitHub MCP binding must not be exposed directly to agents:\n%s", githubBinding)
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

func TestValidateSkillBodySourceBoundaries(t *testing.T) {
	t.Run("requires exactly one body source", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		content := readFile(t, path) + "\n  body: duplicate inline body\n"
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "exactly one of spec.body or spec.bodySource")
	})

	t.Run("requires a body source", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		content := strings.Replace(normalizeNewlines(readFile(t, path)), "  bodySource: create-github-draft-pr/SKILL.md\n", "", 1)
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "exactly one of spec.body or spec.bodySource")
	})

	t.Run("rejects traversal", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		content := strings.Replace(readFile(t, path), "create-github-draft-pr/SKILL.md", "../outside.md", 1)
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "relative path without traversal")
	})

	t.Run("rejects nested traversal", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		content := strings.Replace(readFile(t, path), "create-github-draft-pr/SKILL.md", "create-github-draft-pr/../other/SKILL.md", 1)
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "relative path without traversal")
	})

	t.Run("rejects missing file", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		content := strings.Replace(readFile(t, path), "create-github-draft-pr/SKILL.md", "create-github-draft-pr/MISSING.md", 1)
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "read spec.bodySource")
	})

	t.Run("rejects symlink", func(t *testing.T) {
		packDir := copyPackToTemp(t)
		path := setupBodySourceSkill(t, packDir)
		link := filepath.Join(packDir, "skills/create-github-draft-pr/LINK.md")
		if err := os.Symlink("SKILL.md", link); err != nil {
			t.Fatal(err)
		}
		content := strings.Replace(readFile(t, path), "create-github-draft-pr/SKILL.md", "create-github-draft-pr/LINK.md", 1)
		writeTestFile(t, path, content)
		assertValidateFails(t, packDir, "must be a regular file")
	})
}

func setupBodySourceSkill(t *testing.T, packDir string) string {
	t.Helper()
	path := filepath.Join(packDir, "skills/create-github-draft-pr.yaml")
	writeTestFile(t, path, `$schema: https://actlane.ru/schemas/v1alpha1/skill-contract.schema.json
apiVersion: actlane.ru/v1alpha1
kind: SkillContract
metadata:
  name: create-github-draft-pr
  description: Test external body.
spec:
  bodySource: create-github-draft-pr/SKILL.md
`)
	writeTestFile(t, filepath.Join(packDir, "skills/create-github-draft-pr/SKILL.md"), "External body.\n")
	return path
}

func assertValidateFails(t *testing.T, packDir, want string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Main([]string{"validate", packDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("validate should fail\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("validate error missing %q\nstdout:\n%s\nstderr:\n%s", want, stdout.String(), stderr.String())
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
	assertExists(t, filepath.Join(packDir, "generated/mcp/tools.json"))
	assertExists(t, filepath.Join(packDir, "generated/mcp/server.json"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/broker/broker-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/actlane.lock"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/commands/create-github-draft-pr.md"))
	assertExists(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/AGENTS.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/opencode/.opencode/agents"))
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
		`"actlane-mcp-broker":`,
		`"type": "local"`,
		`"enabled": true`,
		`"actlane"`,
		`"--broker-bundle"`,
		`"./broker/broker-bundle.json"`,
	} {
		if !strings.Contains(snippet, want) {
			t.Fatalf("generated snippet missing %q:\n%s", want, snippet)
		}
	}
	for _, forbidden := range []string{
		`"actlane-pack-author":`,
		`"actlane-safe-gitops":`,
		`"github":`,
		`"docker"`,
		`"--pack"`,
		`"./packs/create-github-draft-pr"`,
	} {
		if strings.Contains(snippet, forbidden) {
			t.Fatalf("generated snippet should not include %q:\n%s", forbidden, snippet)
		}
	}

	command := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/commands/create-github-draft-pr.md"))
	for _, want := range []string{
		`description: "Prepare a safe GitHub draft pull request from reviewed changes."`,
		"Use Actlane capability `create-github-draft-pr`.",
		"$ARGUMENTS",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("generated OpenCode command missing %q:\n%s", want, command)
		}
	}

	skill := readFile(t, filepath.Join(packDir, "generated/opencode/.opencode/skills/create-github-draft-pr/SKILL.md"))
	for _, want := range []string{
		"name: \"create-github-draft-pr\"",
		"description: \"Safely prepare a GitHub draft pull request from reviewed changes.\"",
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
	if strings.Contains(lockfile, "subagents/") || strings.Contains(lockfile, "contracts/") || strings.Contains(lockfile, "runtime-profiles/") {
		t.Fatalf("minimal lockfile contains removed contract sources:\n%s", lockfile)
	}

}

func TestGenerateCodexWritesCodexProfile(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate codex failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	assertExists(t, filepath.Join(packDir, "generated/codex/codex.config.toml"))
	assertExists(t, filepath.Join(packDir, "generated/codex/.agents/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/AGENTS.md"))
	assertExists(t, filepath.Join(packDir, "generated/codex/policies/policy-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/codex/broker/broker-bundle.json"))
	assertExists(t, filepath.Join(packDir, "generated/codex/actlane.lock"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/actlane.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/capabilities"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/policies/github-draft-pr.policy.yaml"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/mcp"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/target-profiles"))
	assertNotExists(t, filepath.Join(packDir, "generated/codex/files"))
	assertNotExists(t, filepath.Join(packDir, ".codex"))

	config := readFile(t, filepath.Join(packDir, "generated/codex/codex.config.toml"))
	for _, want := range []string{
		"[mcp_servers.actlane-mcp-broker]",
		`args = ["mcp", "serve", "--broker-bundle", "./broker/broker-bundle.json"]`,
		`command = "actlane"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("generated Codex config missing %q:\n%s", want, config)
		}
	}
	for _, forbidden := range []string{
		"[mcp_servers.actlane-pack-author]",
		"[mcp_servers.actlane-safe-gitops]",
		"[mcp_servers.github]",
		`command = "docker"`,
		`args = ["mcp", "author", "serve", "--pack", "./packs/create-github-draft-pr"]`,
	} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("generated Codex config should not include %q:\n%s", forbidden, config)
		}
	}

	skill := readFile(t, filepath.Join(packDir, "generated/codex/.agents/skills/create-github-draft-pr/SKILL.md"))
	for _, want := range []string{
		"name: \"create-github-draft-pr\"",
		"description: \"Safely prepare a GitHub draft pull request from reviewed changes.\"",
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

func TestMCPServeListsMinimalBrokerTools(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp serve failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_load_capability"`,
		`"name":"actlane_run_capability"`,
		`"name":"actlane_get_evidence"`,
		`"name":"actlane_prepare_delivery"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{`"name":"actlane_classify"`, `"name":"create_github_draft_pr_audit"`, `"name":"create_github_draft_pr_enforce"`} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("minimal broker should not expose %q:\n%s", forbidden, output)
		}
	}
}

func TestMCPServeLoadsCompactCapabilityView(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_load_capability","arguments":{"name":"create-github-draft-pr"}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp load capability failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_load_capability"`,
		`\"name\": \"create-github-draft-pr\"`,
		`\"policyRef\": \"github-draft-pr-policy\"`,
		`\"executionRef\": \"github-mcp-draft-pr\"`,
		`\"evidenceRef\": \"create-github-draft-pr\"`,
		`\"requiredEvidence\": [`,
		`\"draft_pr_url\"`,
		`\"policy\": {`,
		`\"confirmation\": {`,
		`\"forbidPaths\": [`,
		`\"downstreamTools\": [`,
		`\"create_pull_request\"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp load capability output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		`ghcr.io/github/github-mcp-server`,
		`GITHUB_PERSONAL_ACCESS_TOKEN`,
		`"command": [`,
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("mcp load capability output leaked %q:\n%s", forbidden, output)
		}
	}
}

func TestMCPServeRunsCapabilityThroughPolicyGate(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["README.md"],"confirmed":true}}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","input":{"repo":"unknown/repo","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":[".env"],"confirmed":false}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"actlane_get_evidence","arguments":{"latest":true}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp run capability failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_run_capability"`,
		`"name":"actlane_get_evidence"`,
		`"name":"actlane_prepare_delivery"`,
		`\"capability\": \"create-github-draft-pr\"`,
		`\"policyDecision\": \"allow\"`,
		`\"branch\": \"gpt/feature\"`,
		`\"downstreamPlan\": [`,
		`\"tool\": \"github_create_pull_request\"`,
		`\"adapterSource\": \"MCPBinding\"`,
		`\"adapterExecutions\": [`,
		`\"status\": \"planned\"`,
		`\"binding\": \"github-mcp-draft-pr\"`,
		`\"evidenceId\": \"github-draft-pr-`,
		`\"evidence\": {`,
		`\"contract\": \"create-github-draft-pr\"`,
		`\"rawOutput\": \"summary\"`,
		`\"redacted\": true`,
		`\"policy_decision\": \"allow\"`,
		`\"changed_files\": [`,
		`\"residual_risk\": \"low\"`,
		`\"missingFields\": [`,
		`\"draft_pr_url\"`,
		`\"source\": \"evidenceStore\"`,
		`\"blocked_paths\": [`,
		`\"delivery\": {`,
		`\"summary\": \"Actlane broker prepared create-github-draft-pr with policy decision deny.\"`,
		`\"whatChanged\": [`,
		`\"README.md\"`,
		`\"evidenceId\": \"github-draft-pr-`,
		`\"execution\": {`,
		`\"performed\": false`,
		`\"reason\": \"adapter executions are recorded but external MCP calls are not executed by this MVP\"`,
		`\"policyDecision\": \"deny\"`,
		`"isError":true`,
		`file is forbidden: .env`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp run capability output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		`ghcr.io/github/github-mcp-server`,
		`GITHUB_PERSONAL_ACCESS_TOKEN`,
		`"command": [`,
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("mcp run capability output leaked %q:\n%s", forbidden, output)
		}
	}
}

func TestMCPServeExecutesAdaptersAndPersistsEvidenceWhenExplicitlyEnabled(t *testing.T) {
	if os.Getenv("ACTLANE_FAKE_MCP") == "1" {
		runFakeMCPServer(t)
		return
	}
	packDir := copyPackToTemp(t)
	patchGitHubBindingForFakeMCP(t, packDir)
	evidenceDir := filepath.Join(t.TempDir(), "evidence")
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"name":"create-github-draft-pr","mode":"enforce","executeAdapters":true,"evidenceDir":%q,"input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["README.md"],"confirmed":true}}}}`, evidenceDir),
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"actlane_prepare_delivery","arguments":{"latest":true}}}`,
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"mcp", "serve", "--pack", packDir}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp adapter execution failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_prepare_delivery"`,
		`\"status\": \"succeeded\"`,
		`\"performed\": true`,
		`\"output\": {`,
		`fake-mcp:create_pull_request`,
		`\"durablePath\": \"`,
		`\"externalExecutionDone\": true`,
		`\"policyDecision\": \"allow\"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp adapter execution output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{
		`GITHUB_PERSONAL_ACCESS_TOKEN`,
		`ghcr.io/github/github-mcp-server`,
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("mcp adapter execution output leaked %q:\n%s", forbidden, output)
		}
	}
	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		t.Fatalf("evidence dir missing: %v", err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("expected one durable evidence json file, got %#v", entries)
	}
}

func TestPatchGitHubBindingForFakeMCPSupportsCRLF(t *testing.T) {
	packDir := copyPackToTemp(t)
	path := filepath.Join(packDir, "mcp/bindings/github-mcp-draft-pr.yaml")
	content := strings.ReplaceAll(normalizeNewlines(readFile(t, path)), "\n", "\r\n")
	writeTestFile(t, path, content)

	patchGitHubBindingForFakeMCP(t, packDir)

	patched := readFile(t, path)
	for _, want := range []string{"provider: fake-test-mcp", fmt.Sprintf("- %q", os.Args[0]), "  requiredTools:\n"} {
		if !strings.Contains(patched, want) {
			t.Fatalf("patched binding missing %q:\n%s", want, patched)
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
		"",
	}, "\n"))
	stdout.Reset()
	stderr.Reset()
	code = MainWithIO([]string{"mcp", "serve", "--policy-bundle", filepath.Join(packDir, "generated/codex/policies/policy-bundle.json")}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp serve with policy bundle failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	if strings.Contains(output, `create_github_draft_pr_enforce`) {
		t.Fatalf("minimal policy bundle should not expose generated execution tools:\n%s", output)
	}
}

func TestMCPServeAcceptsBrokerBundle(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}

	bundle := readFile(t, filepath.Join(packDir, "generated/codex/broker/broker-bundle.json"))
	for _, want := range []string{
		`"capabilities":`,
		`"policies":`,
		`"mcpBindings":`,
		`"evidence":`,
	} {
		if !strings.Contains(bundle, want) {
			t.Fatalf("broker bundle missing %q:\n%s", want, bundle)
		}
	}
	for _, forbidden := range []string{
		`"Raw":`,
		`"Path":`,
		`"--pack"`,
		`"SKILL.md"`,
		`"AGENTS.md"`,
		`"opencode.jsonc"`,
	} {
		if strings.Contains(bundle, forbidden) {
			t.Fatalf("broker bundle should not include %q:\n%s", forbidden, bundle)
		}
	}

	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"actlane_run_capability","arguments":{"capability":"create-github-draft-pr","input":{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":["README.md"],"confirmed":true}}}}`,
		"",
	}, "\n"))
	stdout.Reset()
	stderr.Reset()
	code = MainWithIO([]string{"mcp", "serve", "--broker-bundle", filepath.Join(packDir, "generated/codex/broker/broker-bundle.json")}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("mcp serve with broker bundle failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"name":"actlane_prepare_delivery"`,
		`\"create-github-draft-pr\"`,
		`\"policyDecision\": \"allow\"`,
		`\"branch\": \"gpt/feature\"`,
		`\"tool\": \"github_create_pull_request\"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("mcp serve with broker bundle missing %q:\n%s", want, output)
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
		`\"path\": \"generated/codex/.agents/skills/create-github-draft-pr/SKILL.md\"`,
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
	writeTestFile(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"), "user-owned skill\n")
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
		"Will create:",
		".codex/config.toml",
		"policies/policy-bundle.json",
		"broker/broker-bundle.json",
		"source:",
		"preview:",
		"sha256:",
		"Conflicts:",
		".agents/skills/create-github-draft-pr/SKILL.md",
		"Apply blocked: 1 conflict(s)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan output missing %q:\n%s", want, output)
		}
	}
	assertNotExists(t, filepath.Join(projectDir, "broker/broker-bundle.json"))
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
		".agents/skills/create-github-draft-pr/SKILL.md",
		".codex/config.toml",
		"policies/policy-bundle.json",
		"broker/broker-bundle.json",
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
		".agents/skills/create-github-draft-pr/SKILL.md",
		".codex/config.toml",
		"policies/policy-bundle.json",
		"broker/broker-bundle.json",
	} {
		assertExists(t, filepath.Join(projectDir, path))
	}
	config := readFile(t, filepath.Join(projectDir, ".codex/config.toml"))
	for _, want := range []string{
		"# actlane:start github-draft-pr-pack/.codex/config.toml",
		`args = ["mcp", "serve", "--broker-bundle", "./broker/broker-bundle.json"]`,
		"# actlane:end github-draft-pr-pack/.codex/config.toml",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("codex.config.toml missing %q:\n%s", want, config)
		}
	}
	broker := readFile(t, filepath.Join(projectDir, "broker/broker-bundle.json"))
	if !strings.Contains(broker, `"mcpBindings"`) || !strings.Contains(broker, `"github-mcp-draft-pr"`) {
		t.Fatalf("broker bundle should include downstream MCP binding:\n%s", broker)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
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
	assertNotExists(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"))
	assertNotExists(t, filepath.Join(projectDir, ".codex/config.toml"))
	assertNotExists(t, filepath.Join(projectDir, "policies/policy-bundle.json"))
	assertNotExists(t, filepath.Join(projectDir, "broker/broker-bundle.json"))

	writeTestFile(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"), "user-owned\n")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("apply should fail on conflict\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conflicts:") || !strings.Contains(stderr.String(), "apply blocked") {
		t.Fatalf("apply conflict output missing details\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"))
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
	assertExists(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"))

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "codex", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remove failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertNotExists(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"))
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
	assertExists(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"))
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
		`"targetPath": ".agents/skills/create-github-draft-pr/SKILL.md"`,
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
		"+++ .agents/skills/create-github-draft-pr/SKILL.md",
		"+++ .codex/config.toml",
		"content:",
		"name: \"create-github-draft-pr\"",
		"# actlane:start github-draft-pr-pack/.codex/config.toml",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("plan diff output missing %q:\n%s", want, output)
		}
	}
}

func TestPlanOpenCodeSafeAdoptionDetectsUserConfigConflict(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	writeTestFile(t, filepath.Join(projectDir, "opencode.jsonc"), "{\n  // user-owned\n}\n")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{"plan", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("plan failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Plan target: opencode",
		"Will create:",
		".opencode/commands/create-github-draft-pr.md",
		".opencode/skills/create-github-draft-pr/SKILL.md",
		"broker/broker-bundle.json",
		"policies/policy-bundle.json",
		"Conflicts:",
		"opencode.jsonc",
		"Apply blocked: 1 conflict(s)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("OpenCode plan output missing %q:\n%s", want, output)
		}
	}
	assertNotExists(t, filepath.Join(projectDir, ".opencode/commands/create-github-draft-pr.md"))
}

func TestApplyAndRemoveOpenCodeSafeAdoption(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code = Main([]string{"apply", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		"opencode.jsonc",
		".opencode/commands/create-github-draft-pr.md",
		".opencode/skills/create-github-draft-pr/SKILL.md",
		"broker/broker-bundle.json",
		"policies/policy-bundle.json",
	} {
		assertExists(t, filepath.Join(projectDir, path))
	}
	config := readFile(t, filepath.Join(projectDir, "opencode.jsonc"))
	if !strings.Contains(config, `"actlane-mcp-broker"`) || strings.Contains(config, `"github"`) {
		t.Fatalf("OpenCode config must expose only Actlane broker:\n%s", config)
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("second apply failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Skipped:") {
		t.Fatalf("second OpenCode apply should skip unchanged output:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remove failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		"opencode.jsonc",
		".opencode/commands/create-github-draft-pr.md",
		".opencode/skills/create-github-draft-pr/SKILL.md",
		"broker/broker-bundle.json",
		"policies/policy-bundle.json",
	} {
		assertNotExists(t, filepath.Join(projectDir, path))
	}
}

func TestRemoveOpenCodeBlocksUserModifiedGeneratedFile(t *testing.T) {
	packDir := copyPackToTemp(t)
	projectDir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "opencode"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"apply", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("apply failed: %s", stderr.String())
	}
	writeTestFile(t, filepath.Join(projectDir, "opencode.jsonc"), "{\n  // user modified\n}\n")

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"remove", packDir, "--target", "opencode", "--project", projectDir}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("remove should fail on modified OpenCode config\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Conflicts:") || !strings.Contains(stderr.String(), "remove blocked") {
		t.Fatalf("remove conflict output missing details\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(projectDir, "opencode.jsonc"))
	assertExists(t, filepath.Join(projectDir, ".opencode/skills/create-github-draft-pr/SKILL.md"))
}

func TestCheckDeniesWorkflowChanges(t *testing.T) {
	packDir := copyPackToTemp(t)
	stdin := strings.NewReader(`{"repo":"bakaut/development","baseBranch":"main","branch":"feature","title":"Test","summary":"Test","files":[".github/workflows/release.yml"],"confirmed":true}`)
	var stdout, stderr bytes.Buffer

	code := MainWithIO([]string{"check", "--pack", packDir, "--capability", "create-github-draft-pr"}, stdin, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("audit check failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		`"policyDecision": "deny"`,
		`.github/workflows/release.yml`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("check output missing %q:\n%s", want, output)
		}
	}
}

func TestCheckEnforceDeniesSecrets(t *testing.T) {
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
		`secrets/token.txt`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("check enforce output missing %q:\n%s", want, output)
		}
	}
}

func TestGeneratedPolicyBundleCarriesMinimalSafetyPolicy(t *testing.T) {
	packDir := copyPackToTemp(t)
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	bundle := readFile(t, filepath.Join(packDir, "generated/codex/policies/policy-bundle.json"))
	for _, want := range []string{
		`"confirmation":`,
		`"mustBe": true`,
		`"branchPrefix": "gpt/"`,
		`".github/workflows/**"`,
		`"name": "create_pull_request"`,
	} {
		if !strings.Contains(bundle, want) {
			t.Fatalf("policy bundle missing %q:\n%s", want, bundle)
		}
	}
}

func TestGenerateFullPackPreservesSharedMCPArtifacts(t *testing.T) {
	packDir := filepath.Join(repoRoot(t), "packs/full")
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Main([]string{"generate", packDir, "--target", "codex", "--out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate full pack failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(outDir, "generated/mcp/server.json"))
	assertExists(t, filepath.Join(outDir, "generated/mcp/tools.json"))
	config := readFile(t, filepath.Join(outDir, "generated/codex/codex.config.toml"))
	if !strings.Contains(config, "[mcp_servers.actlane-safe-gitops]") {
		t.Fatalf("full pack must preserve agent-facing MCP bindings:\n%s", config)
	}
	tools := readFile(t, filepath.Join(outDir, "generated/mcp/tools.json"))
	for _, want := range []string{`"generatedTool": "create_github_draft_pr_audit"`, `"generatedTool": "create_github_draft_pr_enforce"`} {
		if !strings.Contains(tools, want) {
			t.Fatalf("full pack MCP tools missing %q:\n%s", want, tools)
		}
	}
	policy := readFile(t, filepath.Join(outDir, "generated/codex/policies/policy-bundle.json"))
	if !strings.Contains(policy, `"responsibility":`) {
		t.Fatalf("full pack must preserve responsibility projection:\n%s", policy)
	}
	broker := readFile(t, filepath.Join(outDir, "generated/codex/broker/broker-bundle.json"))
	if !strings.Contains(broker, `"runtimeProfiles":`) {
		t.Fatalf("full pack must preserve runtime profiles:\n%s", broker)
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

	sourcePath := filepath.Join(packDir, "target-profiles/opencode.yaml")
	content := readFile(t, sourcePath)
	writeTestFile(t, sourcePath, strings.Replace(content, "requireDiffPreview: true", "requireDiffPreview: false", 1))

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

func TestFrozenLockfileDetectsSkillBodySourceDrift(t *testing.T) {
	packDir := copyPackToTemp(t)
	setupBodySourceSkill(t, packDir)
	sourcePath := filepath.Join(packDir, "skills/create-github-draft-pr/SKILL.md")
	var stdout, stderr bytes.Buffer

	code := Main([]string{"generate", packDir, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed: %s", stderr.String())
	}
	lockfile := readFile(t, filepath.Join(packDir, "generated/codex/actlane.lock"))
	if !strings.Contains(lockfile, `"skills/create-github-draft-pr/SKILL.md":`) {
		t.Fatalf("lockfile should include skill bodySource digest:\n%s", lockfile)
	}

	content := readFile(t, sourcePath)
	writeTestFile(t, sourcePath, content+"\nBody source drift marker.\n")

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "codex", "--check"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--check should fail after skill bodySource drift")
	}

	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", packDir, "--target", "codex", "--frozen-lockfile"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("--frozen-lockfile should fail after skill bodySource drift")
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
	assertExists(t, filepath.Join(consumerDir, "generated/codex/.agents/skills/create-github-draft-pr/SKILL.md"))
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
	assertExists(t, filepath.Join(installedDir, "generated/codex/.agents/skills/create-github-draft-pr/SKILL.md"))
	assertExists(t, filepath.Join(installedDir, "generated/codex/actlane.lock"))
}

func TestInspectCodexProjectConfig(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "empty-codex-home"))
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "Project Codex guidance.\n")
	writeTestFile(t, filepath.Join(projectDir, ".codex/config.toml"), `
[mcp_servers.github]
command = "github-mcp-server"
args = ["stdio"]
`)
	writeTestFile(t, filepath.Join(projectDir, ".agents/skills/create-github-draft-pr/SKILL.md"), `---
name: create-github-draft-pr
description: Draft PR skill.
---

Use the GitHub draft PR workflow.
`)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"inspect", "--from", projectDir, "--ai-agent", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"ai-agent: codex", "confidence: high", "Project-local:", "skill: create-github-draft-pr", "mcp: github", "Available global objects:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("inspect codex output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestInspectCodexSkillsUsesModernPathsAndLegacyFallback(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	nestedDir := filepath.Join(repoDir, "services", "api")
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "empty-codex-home"))
	writeTestFile(t, filepath.Join(repoDir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeTestFile(t, filepath.Join(repoDir, ".agents/skills/shared/SKILL.md"), `---
name: shared
description: Modern shared skill.
---
Modern body.
`)
	writeTestFile(t, filepath.Join(repoDir, ".codex/skills/shared/SKILL.md"), `---
name: shared
description: Legacy shared skill.
---
Legacy body.
`)
	writeTestFile(t, filepath.Join(nestedDir, ".agents/skills/local/SKILL.md"), `---
name: local
description: Nested local skill.
---
Local body.
`)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"inspect", "--from", nestedDir, "--ai-agent", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"skill: local",
		"skill: shared",
		"Legacy project-local Codex skills found under .codex/skills; prefer .agents/skills.",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("inspect output missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Count(stdout.String(), "skill: shared") != 1 {
		t.Fatalf("modern and legacy duplicate should be deduplicated:\n%s", stdout.String())
	}
	importDir := filepath.Join(t.TempDir(), "imported")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"import", "--from", nestedDir, "--out", importDir, "--ai-agent", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("import failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if skill := readFile(t, filepath.Join(importDir, "skills/shared.yaml")); !strings.Contains(skill, "Modern body.") || strings.Contains(skill, "Legacy body.") {
		t.Fatalf("modern skill must take precedence over legacy duplicate:\n%s", skill)
	}
}
func TestCodexGlobalInventoryAndExplicitImport(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	codexHome := filepath.Join(t.TempDir(), ".codex")
	userHome := t.TempDir()
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("HOME", userHome)
	writeTestFile(t, filepath.Join(projectDir, "AGENTS.md"), "Project Codex guidance.\n")
	writeTestFile(t, filepath.Join(projectDir, ".agents/skills/project-skill/SKILL.md"), `---
name: project-skill
description: Project skill.
---

Project workflow.
`)
	writeTestFile(t, filepath.Join(codexHome, "skills/code-review/SKILL.md"), `---
name: code-review
description: Global review skill.
---

	Review carefully.
`)
	writeTestFile(t, filepath.Join(codexHome, "skills/code-review/references/checklist.md"), "Review checklist.\n")
	writeTestFile(t, filepath.Join(codexHome, "skills/code-review/scripts/check.sh"), "#!/bin/sh\nexit 0\n")
	outsideResource := filepath.Join(t.TempDir(), "outside-secret.txt")
	writeTestFile(t, outsideResource, "must not be imported\n")
	if err := os.Symlink(outsideResource, filepath.Join(codexHome, "skills/code-review/references/outside-secret.txt")); err != nil {
		t.Fatalf("create skill resource symlink: %v", err)
	}
	writeTestFile(t, filepath.Join(codexHome, "skills/incident-response/SKILL.md"), `---
name: incident-response
description: Global incident response skill.
---

Respond carefully.
`)
	writeTestFile(t, filepath.Join(userHome, ".agents/skills/modern-review/SKILL.md"), `---
name: modern-review
description: Modern global review skill.
Review with the modern path.
`)
	writeTestFile(t, filepath.Join(codexHome, "config.toml"), `
[mcp_servers.github]
command = "/usr/local/bin/github-mcp-server"
args = ["stdio"]
env = { GITHUB_TOKEN = "secret-token" }

[mcp_servers.lean-ctx]
command = "lean-ctx"
`)
	writeTestFile(t, filepath.Join(codexHome, "hooks.json"), `{"hooks":{"PreToolUse":[{"hooks":[{"command":"danger"}]}]}}`)

	var stdout, stderr bytes.Buffer
	code := Main([]string{"inspect", "--from", projectDir, "--ai-agent", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inspect failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"Project-local:",
		"skill: project-skill",
		"Available global objects:",
		"skill: code-review [portable candidate]",
		"skill: incident-response [portable candidate]",
		"skill: modern-review [portable candidate]",
		"mcp: github [review required]",
		"hook: pre_tool_use [not portable]",
		"Legacy global Codex skills found under CODEX_HOME/skills; prefer $HOME/.agents/skills.",
		"MCP environment variable values are never transferred",
		"--include-global-skill code-review",
		"--include-global-skill incident-response",
		"--include-global-skill modern-review",
		"--include-global-mcp github",
		"--include-global-mcp lean-ctx",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("inspect output missing %q:\n%s", want, stdout.String())
		}
	}

	localOnly := filepath.Join(t.TempDir(), "local-only")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"import", "--from", projectDir, "--out", localOnly, "--ai-agent", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("local import failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertNotExists(t, filepath.Join(localOnly, "skills/code-review.yaml"))
	assertNotExists(t, filepath.Join(localOnly, "skills/incident-response.yaml"))
	if content := readFile(t, filepath.Join(localOnly, "import.report.md")); strings.Contains(content, "Explicit global imports\n\n- skill: code-review") {
		t.Fatalf("ordinary import must not include global skill:\n%s", content)
	}

	selected := filepath.Join(t.TempDir(), "selected")
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{
		"import", "--from", projectDir, "--out", selected, "--ai-agent", "codex",
		"--include-global-skill", "code-review",
		"--include-global-skill", "incident-response",
		"--include-global-skill", "modern-review",
		"--include-global-mcp=github",
		"--include-global-mcp=lean-ctx",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("selected import failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(selected, "skills/project-skill.yaml"))
	assertExists(t, filepath.Join(selected, "skills/code-review.yaml"))
	assertExists(t, filepath.Join(selected, "skills/incident-response.yaml"))
	assertExists(t, filepath.Join(selected, "skills/modern-review.yaml"))
	assertExists(t, filepath.Join(selected, "skills/code-review/references/checklist.md"))
	assertExists(t, filepath.Join(selected, "skills/code-review/scripts/check.sh"))
	assertNotExists(t, filepath.Join(selected, "skills/code-review/references/outside-secret.txt"))
	binding := readFile(t, filepath.Join(selected, "mcp/bindings/project-skill.yaml"))
	report := readFile(t, filepath.Join(selected, "import.report.md"))
	for _, content := range []string{binding, report} {
		if strings.Contains(content, "secret-token") {
			t.Fatalf("MCP env value leaked:\n%s", content)
		}
		if strings.Contains(content, "danger") {
			t.Fatalf("hook command leaked:\n%s", content)
		}
	}
	for _, want := range []string{"name: github", "/usr/local/bin/github-mcp-server", "name: lean-ctx"} {
		if !strings.Contains(binding, want) {
			t.Fatalf("selected binding missing %q:\n%s", want, binding)
		}
	}
	for _, want := range []string{"skill: code-review", "skill: incident-response", "skill: modern-review", "mcp server: github", "mcp server: lean-ctx", "GITHUB_TOKEN (values excluded)", "hook: pre_tool_use [not portable]"} {
		if !strings.Contains(report, want) {
			t.Fatalf("selected report missing %q:\n%s", want, report)
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"validate", selected}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("selected global import must validate, code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"generate", selected, "--target", "codex"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("selected global import must generate, code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	assertExists(t, filepath.Join(selected, "generated/codex/.agents/skills/code-review/references/checklist.md"))
	assertExists(t, filepath.Join(selected, "generated/codex/.agents/skills/code-review/scripts/check.sh"))
	assertExists(t, filepath.Join(selected, "generated/codex/.agents/skills/incident-response/SKILL.md"))
	assertExists(t, filepath.Join(selected, "generated/codex/.agents/skills/modern-review/SKILL.md"))
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
	assertExists(t, filepath.Join(outDir, "generated/codex/.agents/skills/safe-deploy/SKILL.md"))

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

func patchGitHubBindingForFakeMCP(t *testing.T, packDir string) {
	t.Helper()
	path := filepath.Join(packDir, "mcp/bindings/github-mcp-draft-pr.yaml")
	content := normalizeNewlines(readFile(t, path))
	start := strings.Index(content, "  mcpservers:\n")
	end := strings.Index(content, "  requiredTools:\n")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("cannot locate mcpservers block in %s", path)
	}
	replacement := fmt.Sprintf(`  mcpservers:
    - name: github
      provider: fake-test-mcp
      source: test-helper
      transport: stdio
      command:
        - %q
      args:
        - -test.run=TestMCPServeExecutesAdaptersAndPersistsEvidenceWhenExplicitlyEnabled
        - --
      env:
        ACTLANE_FAKE_MCP: "1"
`, os.Args[0])
	if err := os.WriteFile(path, []byte(content[:start]+replacement+content[end:]), 0o644); err != nil {
		t.Fatal(err)
	}
}

func normalizeNewlines(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}

func runFakeMCPServer(t *testing.T) {
	t.Helper()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      any             `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintln(os.Stdout, `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`)
			continue
		}
		switch req.Method {
		case "initialize":
			writeFakeMCPResponse(t, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo":      map[string]any{"name": "fake-github-mcp", "version": "test"},
			})
		case "tools/call":
			var params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				writeFakeMCPError(t, req.ID, -32602, "invalid params")
				continue
			}
			writeFakeMCPResponse(t, req.ID, map[string]any{
				"content": []map[string]any{{
					"type": "text",
					"text": "fake-mcp:" + params.Name,
				}},
				"structuredContent": map[string]any{
					"tool":   params.Name,
					"branch": params.Arguments["branch"],
				},
			})
		default:
			writeFakeMCPError(t, req.ID, -32601, "method not found")
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func writeFakeMCPResponse(t *testing.T, id any, result any) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, string(data))
}

func writeFakeMCPError(t *testing.T, id any, code int, message string) {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintln(os.Stdout, string(data))
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
