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
	Name      string
	Targets   []string
	Contracts []string
}

type ContractSet map[string]bool

const (
	ContractCapability     = "capability"
	ContractPolicy         = "policy"
	ContractMCPBinding     = "mcp"
	ContractSkill          = "skill"
	ContractCommand        = "command"
	ContractAgent          = "agent"
	ContractResponsibility = "responsibility"
	ContractRuntimeProfile = "runtime-profile"
	ContractEvidence       = "evidence"
	ContractTargetProfile  = "target-profile"
)

func Plan(opts Options) ([]File, error) {
	name := CleanName(opts.Name)
	targets := cleanTargets(opts.Targets)
	contracts, err := NormalizeContracts(opts.Contracts)
	if err != nil {
		return nil, err
	}
	files := []File{
		{
			Path:    "actlane.yaml",
			Purpose: "CapabilityPack manifest that references source contracts.",
			Content: packManifestProposal(name, targets, contracts),
		},
	}
	if contracts.Has(ContractCapability) {
		files = append(files, File{
			Path:    "capabilities/" + name + ".yaml",
			Purpose: "Capability source of truth for intent, interface, policy refs, MCP binding refs, and reporting.",
			Content: capabilityProposal(name, contracts),
		})
	}
	if contracts.Has(ContractPolicy) {
		files = append(files, File{
			Path:    "policies/" + name + "-policy.yaml",
			Purpose: "Safety policy for allow, deny, mutation, limits, and approval behavior.",
			Content: policyProposal(name),
		})
	}
	if contracts.Has(ContractMCPBinding) {
		files = append(files, File{
			Path:    "mcp/bindings/" + name + ".yaml",
			Purpose: "Runtime MCP binding and real downstream tool mapping.",
			Content: mcpBindingProposal(name),
		})
	}
	if contracts.Has(ContractSkill) {
		files = append(files, File{
			Path:    "skills/" + name + ".yaml",
			Purpose: "Portable skill directory contract: SKILL.md body plus optional resources.",
			Content: skillProposal(name),
		})
	}
	if contracts.Has(ContractCommand) {
		files = append(files, File{
			Path:    "commands/" + name + ".yaml",
			Purpose: "Portable command contract for slash command entrypoints.",
			Content: commandProposal(name, contracts),
		})
	}
	if contracts.Has(ContractAgent) {
		files = append(files, File{
			Path:    "agents/" + name + ".yaml",
			Purpose: "Portable agent contract for role, activation, capability, and skill references.",
			Content: agentProposal(name, contracts),
		})
	}
	if contracts.Has(ContractResponsibility) {
		files = append(files, File{
			Path:    "contracts/" + name + ".yaml",
			Purpose: "Responsibility contract for risk, evidence, and human approval semantics.",
			Content: responsibilityProposal(name),
		})
	}
	if contracts.Has(ContractRuntimeProfile) {
		files = append(files, File{
			Path:    "runtime-profiles/" + name + ".yaml",
			Purpose: "Runtime routing contract for work type, risk flags, mode, and candidate capabilities.",
			Content: runtimeProfileProposal(name),
		})
	}
	if contracts.Has(ContractEvidence) {
		files = append(files, File{
			Path:    "evidence/" + name + ".yaml",
			Purpose: "Compact evidence contract for required summary fields and raw-output boundary.",
			Content: evidenceProposal(name),
		})
	}
	for _, target := range targets {
		if contracts.Has(ContractTargetProfile) {
			files = append(files, File{
				Path:    "target-profiles/" + target + ".yaml",
				Purpose: "Target runtime file layout and generated file mapping.",
				Content: targetProfileProposal(target, name, contracts),
			})
		}
	}
	return files, nil
}

func (contracts ContractSet) Has(name string) bool {
	return contracts[name]
}

func DefaultContracts() []string {
	return []string{
		ContractCapability,
		ContractPolicy,
		ContractMCPBinding,
		ContractSkill,
		ContractTargetProfile,
	}
}

func AllContracts() []string {
	return []string{
		ContractCapability,
		ContractPolicy,
		ContractMCPBinding,
		ContractSkill,
		ContractCommand,
		ContractAgent,
		ContractResponsibility,
		ContractRuntimeProfile,
		ContractEvidence,
		ContractTargetProfile,
	}
}

func NormalizeContracts(values []string) (ContractSet, error) {
	if len(values) == 0 {
		values = DefaultContracts()
	}
	contracts := ContractSet{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			contract, err := normalizeContract(part)
			if err != nil {
				return nil, err
			}
			if contract == "default" {
				for _, item := range DefaultContracts() {
					contracts[item] = true
				}
				continue
			}
			if contract == "all" {
				for _, item := range AllContracts() {
					contracts[item] = true
				}
				continue
			}
			contracts[contract] = true
		}
	}
	for _, required := range []string{ContractCapability, ContractPolicy, ContractTargetProfile} {
		if !contracts.Has(required) {
			return nil, fmt.Errorf("contracts must include %s for a valid pack", required)
		}
	}
	return contracts, nil
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

func normalizeContract(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	switch value {
	case "":
		return "", fmt.Errorf("contract name is required")
	case "default":
		return "default", nil
	case "all":
		return "all", nil
	case "capability", "capabilities":
		return ContractCapability, nil
	case "policy", "policies", "tool-call-policy", "toolcallpolicy":
		return ContractPolicy, nil
	case "mcp", "mcp-binding", "mcp-bindings", "mcpbinding", "mcpbindings":
		return ContractMCPBinding, nil
	case "skill", "skills", "skill-contract", "skill-contracts", "skillcontract", "skillcontracts":
		return ContractSkill, nil
	case "command", "commands", "command-contract", "command-contracts", "commandcontract", "commandcontracts":
		return ContractCommand, nil
	case "agent", "agents", "subagent", "subagents", "agent-contract", "agent-contracts", "agentcontract", "agentcontracts":
		return ContractAgent, nil
	case "responsibility", "responsibilities", "responsibility-contract", "responsibility-contracts", "responsibilitycontract", "responsibilitycontracts", "contract", "contracts":
		return ContractResponsibility, nil
	case "runtime", "runtime-profile", "runtime-profiles", "runtimeprofile", "runtimeprofiles":
		return ContractRuntimeProfile, nil
	case "evidence", "evidence-contract", "evidence-contracts", "evidencecontract", "evidencecontracts":
		return ContractEvidence, nil
	case "target-profile", "target-profiles", "targetprofile", "targetprofiles", "profile", "profiles":
		return ContractTargetProfile, nil
	default:
		return "", fmt.Errorf("unknown contract %q", value)
	}
}

func packManifestProposal(name string, targets []string, contracts ContractSet) string {
	var b strings.Builder
	b.WriteString("apiVersion: actlane.ru/v1alpha1\n")
	b.WriteString("kind: CapabilityPack\n")
	b.WriteString("metadata:\n")
	b.WriteString("  name: " + name + "-pack\n")
	b.WriteString("  version: 0.1.0-alpha.1\n")
	b.WriteString("  description: TODO describe this Actlane pack.\n")
	b.WriteString("spec:\n")
	if contracts.Has(ContractCapability) {
		b.WriteString("  capabilities:\n")
		b.WriteString("    - capabilities/" + name + ".yaml\n")
	}
	if contracts.Has(ContractPolicy) {
		b.WriteString("  policies:\n")
		b.WriteString("    - policies/" + name + "-policy.yaml\n")
	}
	if contracts.Has(ContractMCPBinding) {
		b.WriteString("  mcpBindings:\n")
		b.WriteString("    - mcp/bindings/" + name + ".yaml\n")
	}
	if contracts.Has(ContractSkill) {
		b.WriteString("  skills:\n")
		b.WriteString("    - skills/" + name + ".yaml\n")
	}
	if contracts.Has(ContractCommand) {
		b.WriteString("  commands:\n")
		b.WriteString("    - commands/" + name + ".yaml\n")
	}
	if contracts.Has(ContractAgent) {
		b.WriteString("  agents:\n")
		b.WriteString("    - agents/" + name + ".yaml\n")
	}
	if contracts.Has(ContractResponsibility) {
		b.WriteString("  contracts:\n")
		b.WriteString("    - contracts/" + name + ".yaml\n")
	}
	if contracts.Has(ContractRuntimeProfile) {
		b.WriteString("  runtimeProfiles:\n")
		b.WriteString("    - runtime-profiles/" + name + ".yaml\n")
	}
	if contracts.Has(ContractEvidence) {
		b.WriteString("  evidence:\n")
		b.WriteString("    - evidence/" + name + ".yaml\n")
	}
	if contracts.Has(ContractTargetProfile) {
		b.WriteString("  targetProfiles:\n")
		for _, target := range targets {
			b.WriteString("    - target-profiles/" + target + ".yaml\n")
		}
	}
	b.WriteString("  targets:\n")
	for _, target := range targets {
		b.WriteString("    - " + target + "\n")
	}
	return b.String()
}

func capabilityProposal(name string, contracts ContractSet) string {
	var refs strings.Builder
	refs.WriteString("  policyRef:\n")
	refs.WriteString("    name: " + name + "-policy\n")
	if contracts.Has(ContractMCPBinding) {
		refs.WriteString("  executionRef:\n")
		refs.WriteString("    name: " + name + "\n")
	}
	if contracts.Has(ContractResponsibility) {
		refs.WriteString("  responsibilityRef:\n")
		refs.WriteString("    name: " + name + "\n")
	}
	if contracts.Has(ContractRuntimeProfile) {
		refs.WriteString("  runtimeRef:\n")
		refs.WriteString("    name: " + name + "\n")
	}
	if contracts.Has(ContractEvidence) {
		refs.WriteString("  evidenceRef:\n")
		refs.WriteString("    name: " + name + "\n")
	}
	if contracts.Has(ContractMCPBinding) {
		refs.WriteString("  projections:\n")
		refs.WriteString("    mcp:\n")
		refs.WriteString("      enabled: true\n")
	}
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
%[2]s
  inputs:
    request:
      type: object
      required: true
  outputs:
    result:
      type: object
      required: true
  reporting:
    policyDecision: true
`, name, strings.TrimRight(refs.String(), "\n"))
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

func commandProposal(name string, contracts ContractSet) string {
	var agentRef string
	if contracts.Has(ContractAgent) {
		agentRef = fmt.Sprintf(`  agentRef:
    name: %[1]s
`, name)
	}
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: CommandContract
metadata:
  name: %[1]s
  description: TODO describe the slash command entrypoint.
spec:
  scope: project
  invocation:
    slash: /%[1]s
  capabilityRef:
    name: %[1]s
%[2]s  arguments:
    mode: passthrough
    placeholder: "{{ arguments }}"
    description: TODO describe command arguments.
  prompt:
    template: |
      Use the %[1]s capability.

      Request:
      {{ arguments }}
`, name, agentRef)
}

func agentProposal(name string, contracts ContractSet) string {
	var skills string
	if contracts.Has(ContractSkill) {
		skills = fmt.Sprintf(`  skills:
    allowed:
      - %[1]s
`, name)
	}
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: AgentContract
metadata:
  name: %[1]s
  description: TODO describe this specialized agent.
spec:
  scope: project
  mode: subagent
  role:
    summary: TODO describe the agent role and boundaries.
  activation:
    whenToUse:
      - TODO describe when this agent should be selected.
  capabilities:
    allowed:
      - %[1]s
%[2]s  tools:
    strategy: use-capability-contracts
`, name, skills)
}

func responsibilityProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: ResponsibilityContract
metadata:
  name: %[1]s
  description: TODO describe the responsibility boundary.
spec:
  purpose: TODO describe what this capability is responsible for.
  risks:
    - TODO describe the primary risk.
  evidence:
    required:
      - TODO describe required evidence.
  humanApproval:
    required: false
`, name)
}

func runtimeProfileProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: RuntimeProfile
metadata:
  name: %[1]s
  description: TODO describe runtime classification for this capability.
spec:
  capabilityRef:
    name: %[1]s
  defaultMode: advise
  workTypes:
    - docs_change
    - code_change
    - config_change
    - unknown_or_mixed
  riskFlags:
    - secrets_sensitive
    - destructive_operation
    - production_sensitive
    - security_sensitive
  techHints: []
  candidateCapabilities:
    - %[1]s
  highRisk:
    mode: read-only
    requireHumanBoundary: true
    flags:
      - secrets_sensitive
      - destructive_operation
      - production_sensitive
  recommendations:
    nextStep: Continue with the existing workflow and use Actlane policy checks before mutation.
    humanBoundaryNextStep: Stop before mutation, report the risk, and ask for explicit human boundary approval.
`, name)
}

func evidenceProposal(name string) string {
	return fmt.Sprintf(`apiVersion: actlane.ru/v1alpha1
kind: EvidenceContract
metadata:
  name: %[1]s
  description: TODO describe compact evidence required for this capability.
spec:
  capabilityRef:
    name: %[1]s
  categories:
    - policy
    - risk
  summaryFields:
    - policy_decision
    - changed_files
    - branch
    - residual_risk
  rawOutput:
    default: summary
  redaction:
    secrets: true
    tokens: true
  deliveryChecklist:
    - Report the policy decision.
    - Report changed files.
    - Report residual risk and human approval needs.
  evidenceId:
    prefix: %[1]s
`, name)
}

func targetProfileProposal(target, name string, contracts ContractSet) string {
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
%[2]s`, name, targetProfileFileMappings("opencode", name, contracts))
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
%[2]s`, name, targetProfileFileMappings("codex", name, contracts))
	}
}

func targetProfileFileMappings(target, name string, contracts ContractSet) string {
	var b strings.Builder
	if contracts.Has(ContractCommand) && target == "opencode" {
		b.WriteString(fmt.Sprintf(`      - targetPath: .opencode/commands/%[1]s.md
        generatedPath: generated/opencode/.opencode/commands/%[1]s.md
        commandContract: %[1]s
        owned: true
`, name))
	}
	if contracts.Has(ContractAgent) && target == "opencode" {
		b.WriteString(fmt.Sprintf(`      - targetPath: .opencode/agents/%[1]s.md
        generatedPath: generated/opencode/.opencode/agents/%[1]s.md
        agentContract: %[1]s
        owned: true
`, name))
	}
	if contracts.Has(ContractSkill) {
		switch target {
		case "opencode":
			b.WriteString(fmt.Sprintf(`      - targetPath: .opencode/skills/%[1]s/SKILL.md
        generatedPath: generated/opencode/.opencode/skills/%[1]s/SKILL.md
        skillContract: %[1]s
        owned: true
`, name))
		default:
			b.WriteString(fmt.Sprintf(`      - targetPath: .agents/skills/%[1]s/SKILL.md
        generatedPath: generated/codex/.agents/skills/%[1]s/SKILL.md
        skillContract: %[1]s
        owned: true
`, name))
		}
	}
	return b.String()
}
