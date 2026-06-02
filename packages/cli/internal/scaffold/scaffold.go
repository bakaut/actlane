package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
	Content string `json:"content"`
}

type Options struct {
	Name    string
	Targets []string
}

func Plan(opts Options) []File {
	name := CleanName(opts.Name)
	targets := cleanTargets(opts.Targets)
	files := []File{
		{
			Path:    "actlane.yaml",
			Purpose: "CapabilityPack manifest that references source contracts.",
			Content: packManifestProposal(name, targets),
		},
		{
			Path:    "capabilities/" + name + ".yaml",
			Purpose: "Capability source of truth for intent, interface, policy refs, MCP binding refs, and reporting.",
			Content: capabilityProposal(name),
		},
		{
			Path:    "policies/" + name + "-policy.yaml",
			Purpose: "Safety policy for allow, deny, mutation, limits, and approval behavior.",
			Content: policyProposal(name),
		},
		{
			Path:    "mcp/bindings/" + name + ".yaml",
			Purpose: "Runtime MCP binding and real downstream tool mapping.",
			Content: mcpBindingProposal(name),
		},
		{
			Path:    "skills/" + name + ".yaml",
			Purpose: "Portable skill directory contract: SKILL.md body plus optional resources.",
			Content: skillProposal(name),
		},
	}
	for _, target := range targets {
		files = append(files, File{
			Path:    "target-profiles/" + target + ".yaml",
			Purpose: "Target runtime file layout and generated file mapping.",
			Content: targetProfileProposal(target, name),
		})
	}
	return files
}

func Write(root string, files []File, force bool) ([]string, []string, error) {
	var skipped []string
	safePaths := make([]string, 0, len(files))
	for _, file := range files {
		rel, err := SafeSourcePath(file.Path)
		if err != nil {
			return nil, nil, err
		}
		safePaths = append(safePaths, rel)
		target := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(target); err == nil && !force {
			skipped = append(skipped, rel)
		} else if err != nil && !os.IsNotExist(err) {
			return nil, nil, err
		}
	}
	if len(skipped) > 0 {
		sort.Strings(skipped)
		return nil, skipped, fmt.Errorf("one or more files already exist")
	}
	var written []string
	for i, file := range files {
		rel := safePaths[i]
		target := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return written, nil, err
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return written, nil, err
		}
		written = append(written, rel)
	}
	sort.Strings(written)
	return written, nil, nil
}

func SafeSourcePath(value string) (string, error) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("file path is required")
	}
	if strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("absolute paths are not allowed: %s", value)
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("path traversal is not allowed: %s", value)
	}
	if strings.HasPrefix(cleaned, "generated/") || cleaned == "generated" {
		return "", fmt.Errorf("scaffold must not write generated output: %s", value)
	}
	if strings.HasPrefix(cleaned, ".git/") || cleaned == ".git" {
		return "", fmt.Errorf("scaffold must not write git internals: %s", value)
	}
	return cleaned, nil
}

func CleanName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return "new-capability"
	}
	return cleaned
}

func cleanTargets(values []string) []string {
	if len(values) == 0 {
		return []string{"codex"}
	}
	seen := map[string]bool{}
	var targets []string
	for _, value := range values {
		target := CleanName(value)
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		targets = append(targets, target)
	}
	if len(targets) == 0 {
		return []string{"codex"}
	}
	return targets
}

func packManifestProposal(name string, targets []string) string {
	var b strings.Builder
	b.WriteString("apiVersion: actlane.ru/v1alpha1\n")
	b.WriteString("kind: CapabilityPack\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + name + "-pack\n")
	b.WriteString("  version: 0.1.0-alpha.1\n")
	b.WriteString("  description: TODO describe this Actlane pack.\n")
	b.WriteString("spec:\n")
	b.WriteString("  capabilities:\n")
	b.WriteString("    - capabilities/" + name + ".yaml\n")
	b.WriteString("  policies:\n")
	b.WriteString("    - policies/" + name + "-policy.yaml\n")
	b.WriteString("  mcpBindings:\n")
	b.WriteString("    - mcp/bindings/" + name + ".yaml\n")
	b.WriteString("  skills:\n")
	b.WriteString("    - skills/" + name + ".yaml\n")
	b.WriteString("  targetProfiles:\n")
	for _, target := range targets {
		b.WriteString("    - target-profiles/" + target + ".yaml\n")
	}
	b.WriteString("  targets:\n")
	for _, target := range targets {
		b.WriteString("    - " + target + "\n")
	}
	return b.String()
}

func capabilityProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: Capability
metadata:
  name: %[1]s
  description: TODO describe the safe action.
spec:
  intent:
    type: TODO
    whenToUse:
      - TODO
    whenNotToUse:
      - TODO
  policyRef:
    name: %[1]s-policy
  executionRef:
    name: %[1]s
  inputs:
    request:
      type: object
      required: true
  outputs:
    result:
      type: object
      required: true
  projections:
    mcp:
      enabled: true
  reporting:
    policyDecision: true
`, name)
}

func policyProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: ToolCallPolicy
metadata:
  name: %[1]s-policy
spec:
  match:
    capabilities:
      - %[1]s
  validate:
    confirmation:
      field: confirmed
      mustBe: true
`, name)
}

func mcpBindingProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: MCPBinding
metadata:
  name: %[1]s
spec:
  capabilityRef:
    name: %[1]s
  mcpservers:
    - name: %[1]s
      provider: TODO
      source: local
      transport: stdio
      command:
        - TODO
  requiredTools: []
  strategy:
    type: builtin
    handler: actlane.policy.evaluate
`, name)
}

func skillProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: SkillContract
metadata:
  name: %[1]s
  description: TODO describe when this skill should be used.
spec:
  body: |
    TODO write agent-facing workflow instructions.
`, name)
}

func targetProfileProposal(target, name string) string {
	target = CleanName(target)
	switch target {
	case "opencode":
		return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: TargetProfile
metadata:
  name: opencode
spec:
  target: opencode
  scope: project
  output:
    root: generated/opencode
    config: opencode.jsonc
  generate:
    config: true
    skills: true
    mcp: true
  opencode:
    config:
      filename: opencode.jsonc
      schema: https://opencode.ai/config.json
    files:
      - targetPath: .opencode/skills/%[1]s/SKILL.md
        generatedPath: generated/opencode/.opencode/skills/%[1]s/SKILL.md
        skillContract: %[1]s
        owned: true
`, name)
	default:
		return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: TargetProfile
metadata:
  name: codex
spec:
  target: codex
  scope: project
  output:
    root: generated/codex
    config: codex.config.toml
  install:
    mode: manual-copy
    scope: project
    requireExplicitApply: true
    requireDiffPreview: true
  generate:
    config: true
    skills: true
    mcp: true
  codex:
    config:
      filename: codex.config.toml
    files:
      - targetPath: .codex/skills/%[1]s/SKILL.md
        generatedPath: generated/codex/.codex/skills/%[1]s/SKILL.md
        skillContract: %[1]s
        owned: true
`, name)
	}
}
