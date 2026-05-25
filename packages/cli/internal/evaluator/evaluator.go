package evaluator

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type Request struct {
	Tool       string
	Mode       string
	Capability string
	Input      map[string]any
}

type Evaluation struct {
	Tool                  string         `json:"tool,omitempty"`
	Mode                  string         `json:"mode,omitempty"`
	Capability            string         `json:"capability"`
	PolicyDecision        string         `json:"policyDecision"`
	Allowed               bool           `json:"allowed"`
	Risk                  string         `json:"risk,omitempty"`
	ImpactedScopes        []string       `json:"impactedScopes,omitempty"`
	RequiredChecks        []string       `json:"requiredChecks,omitempty"`
	RequiredEvidence      []string       `json:"requiredEvidence,omitempty"`
	HumanApprovalRequired bool           `json:"humanApprovalRequired,omitempty"`
	Stop                  bool           `json:"stop,omitempty"`
	Reasons               []string       `json:"reasons,omitempty"`
	OriginalInput         map[string]any `json:"originalInput"`
	MutatedInput          map[string]any `json:"mutatedInput"`
	Next                  []NextCall     `json:"next,omitempty"`
	PolicyRefs            []string       `json:"policyRefs"`
	ResponsibilityRefs    []string       `json:"responsibilityRefs,omitempty"`
}

type NextCall struct {
	Server    string         `json:"server"`
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type riskState struct {
	Risk                  string
	Scopes                []string
	RequiredChecks        []string
	RequiredEvidence      []string
	HumanApprovalRequired bool
	Stop                  bool
	Reasons               []string
	Refs                  []string
}

func Evaluate(loaded *pack.LoadedPack, req Request) Evaluation {
	original := cloneMap(req.Input)
	mutated := cloneMap(req.Input)
	policies := policiesFor(loaded, req.Capability)
	reasons := mutate(mutated, policies)
	reasons = append(reasons, validate(mutated, policies)...)
	risk := evaluateResponsibility(loaded, req.Capability, mutated)
	reasons = append(reasons, risk.Reasons...)

	decision := "allow"
	allowed := len(reasons) == 0
	if risk.Stop {
		decision = "deny"
		allowed = false
	} else if len(reasons) > 0 {
		decision = "deny"
		allowed = false
	} else if risk.HumanApprovalRequired {
		decision = "requires_approval"
		allowed = false
	}

	eval := Evaluation{
		Tool:                  req.Tool,
		Mode:                  req.Mode,
		Capability:            req.Capability,
		PolicyDecision:        decision,
		Allowed:               allowed,
		Risk:                  risk.Risk,
		ImpactedScopes:        risk.Scopes,
		RequiredChecks:        risk.RequiredChecks,
		RequiredEvidence:      risk.RequiredEvidence,
		HumanApprovalRequired: risk.HumanApprovalRequired,
		Stop:                  risk.Stop,
		Reasons:               reasons,
		OriginalInput:         original,
		MutatedInput:          mutated,
		PolicyRefs:            policyNames(policies),
		ResponsibilityRefs:    risk.Refs,
	}
	if allowed {
		eval.Next = nextCalls(loaded, req.Capability, mutated)
	}
	return eval
}

func policiesFor(loaded *pack.LoadedPack, capability string) []pack.Policy {
	var policies []pack.Policy
	for _, policy := range loaded.Policies {
		for _, name := range policy.Spec.Match.Capabilities {
			if name == capability {
				policies = append(policies, policy)
				break
			}
		}
	}
	return policies
}

func mutate(input map[string]any, policies []pack.Policy) []string {
	var reasons []string
	for _, policy := range policies {
		for key, value := range policy.Spec.Mutate.Defaults {
			if _, exists := input[key]; !exists {
				input[key] = value
			}
		}
		prefix := policy.Spec.Mutate.Ensure.BranchPrefix
		if prefix != "" {
			branch, _ := input["branch"].(string)
			if branch == "" {
				reasons = append(reasons, "branch is required")
				continue
			}
			if !strings.HasPrefix(branch, prefix) {
				input["branch"] = prefix + branch
			}
		}
	}
	return reasons
}

func validate(input map[string]any, policies []pack.Policy) []string {
	var reasons []string
	for _, policy := range policies {
		confirmation := policy.Spec.Validate.Confirmation
		if confirmation.Field != "" && input[confirmation.Field] != confirmation.MustBe {
			reasons = append(reasons, fmt.Sprintf("%s must be %v", confirmation.Field, confirmation.MustBe))
		}
		if len(policy.Spec.Validate.RepoAllowlist) > 0 {
			repo, _ := input["repo"].(string)
			if !contains(policy.Spec.Validate.RepoAllowlist, repo) {
				reasons = append(reasons, "repo is not allowlisted")
			}
		}
		files := stringSlice(input["files"])
		for _, file := range files {
			if matchPattern(file, policy.Spec.Validate.ForbidPaths) {
				reasons = append(reasons, "file is forbidden: "+file)
			}
		}
		if policy.Spec.Validate.Limits.MaxFiles > 0 && len(files) > policy.Spec.Validate.Limits.MaxFiles {
			reasons = append(reasons, "too many files")
		}
		if policy.Spec.Validate.Limits.MaxDiffKB > 0 {
			if diffKB, ok := number(input["diffKb"]); ok && int(diffKB) > policy.Spec.Validate.Limits.MaxDiffKB {
				reasons = append(reasons, "diff is too large")
			}
		}
	}
	return reasons
}

func evaluateResponsibility(loaded *pack.LoadedPack, capability string, input map[string]any) riskState {
	state := riskState{Risk: "low"}
	files := changedFiles(input)
	for _, contract := range loaded.Contracts {
		if contract.Metadata.Name != "" && contract.Metadata.Name != capability {
			continue
		}
		state.Refs = append(state.Refs, contract.Metadata.Name)
		contractRisk, scopes := classifyRisk(contract.Spec, files)
		state.Risk = maxRisk(state.Risk, contractRisk)
		state.Scopes = appendUnique(state.Scopes, scopes...)
		state.RequiredEvidence = appendUnique(state.RequiredEvidence, stringSliceFromPath(contract.Spec, "evidence", "requiredForHandoff")...)
		state.Reasons = append(state.Reasons, toolGovernanceReasons(contract.Spec, loaded, input)...)
	}
	if state.Risk == "" {
		state.Risk = "low"
	}
	for _, contract := range loaded.Contracts {
		if contract.Metadata.Name != "" && contract.Metadata.Name != capability {
			continue
		}
		class := mapFromPath(contract.Spec, "risk", "classes", state.Risk)
		state.RequiredChecks = appendUnique(state.RequiredChecks, stringSliceFromMap(class, "requiredChecks")...)
		state.HumanApprovalRequired = state.HumanApprovalRequired || boolFromMap(class, "humanApprovalRequired")
		state.Stop = state.Stop || boolFromMap(class, "agentMustStop")
		state.RequiredChecks = appendUnique(state.RequiredChecks, checksForRisk(contract.Spec, state.Risk)...)
	}
	if checkEvidence, _ := input["checkEvidence"].(bool); checkEvidence {
		state.Reasons = append(state.Reasons, missingEvidenceReasons(input, state.RequiredEvidence)...)
	}
	sort.Strings(state.Scopes)
	return state
}

func classifyRisk(spec map[string]any, files []string) (string, []string) {
	risk := "low"
	var scopes []string
	for _, scope := range mapSlice(spec["scopes"]) {
		name := stringFromMap(scope, "name")
		scopeRisk := stringFromMap(scope, "riskFloor")
		patterns := stringSliceFromMap(scope, "paths")
		for _, file := range files {
			if matchPattern(file, patterns) {
				risk = maxRisk(risk, scopeRisk)
				scopes = appendUnique(scopes, name)
				break
			}
		}
	}
	if len(files) > 0 && len(scopes) == 0 {
		risk = "medium"
	}
	return risk, scopes
}

func checksForRisk(spec map[string]any, risk string) []string {
	var checks []string
	required := mapFromPath(spec, "checks")
	for _, check := range mapSlice(required["required"]) {
		if contains(stringSliceFromMap(check, "requiredFor"), risk) {
			checks = append(checks, stringFromMap(check, "name"))
		}
	}
	return checks
}

func toolGovernanceReasons(spec map[string]any, loaded *pack.LoadedPack, input map[string]any) []string {
	mcp := mapFromPath(spec, "tools", "mcp")
	if !boolFromMap(mcp, "denyUnregisteredWriteTools") {
		return nil
	}
	registered := map[string]bool{}
	for _, binding := range loaded.MCPBindings {
		for _, tool := range binding.Spec.RequiredTools {
			registered[tool.Server+"_"+tool.Name] = true
			registered[tool.Name] = true
		}
	}
	var reasons []string
	for _, tool := range stringSlice(input["toolsRequested"]) {
		if registered[tool] || !looksLikeWriteTool(tool) {
			continue
		}
		reasons = append(reasons, "unregistered write MCP tool: "+tool)
	}
	return reasons
}

func missingEvidenceReasons(input map[string]any, required []string) []string {
	evidence, _ := input["evidence"].(map[string]any)
	var reasons []string
	for _, name := range required {
		if evidence[name] == nil {
			reasons = append(reasons, "missing evidence: "+name)
		}
	}
	return reasons
}

func looksLikeWriteTool(name string) bool {
	for _, marker := range []string{"create", "update", "delete", "push", "merge", "apply", "write"} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func nextCalls(loaded *pack.LoadedPack, capability string, input map[string]any) []NextCall {
	var calls []NextCall
	for _, binding := range loaded.MCPBindings {
		if binding.Spec.CapabilityRef.Name != capability || binding.Spec.Strategy.Handler == "actlane.policy.evaluate" {
			continue
		}
		for _, tool := range binding.Spec.RequiredTools {
			calls = append(calls, NextCall{
				Server:    tool.Server,
				Tool:      tool.Server + "_" + tool.Name,
				Arguments: cloneMap(input),
			})
		}
	}
	return calls
}

func changedFiles(input map[string]any) []string {
	files := stringSlice(input["changedFiles"])
	if len(files) == 0 {
		files = stringSlice(input["files"])
	}
	return files
}

func cloneMap(input map[string]any) map[string]any {
	clone := map[string]any{}
	for key, value := range input {
		clone[key] = value
	}
	return clone
}

func policyNames(policies []pack.Policy) []string {
	names := make([]string, 0, len(policies))
	for _, policy := range policies {
		names = append(names, policy.Metadata.Name)
	}
	return names
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if ok {
			values = append(values, text)
		}
	}
	return values
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
}

func matchPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern == file {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(file, strings.TrimSuffix(pattern, "/**")+"/") {
			return true
		}
		if strings.HasPrefix(pattern, "**/*") {
			needle := strings.Trim(strings.TrimPrefix(pattern, "**/"), "*")
			if needle != "" && strings.Contains(file, needle) {
				return true
			}
		}
		if ok, _ := path.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

func maxRisk(a, b string) string {
	if riskRank(b) > riskRank(a) {
		return b
	}
	return a
}

func riskRank(risk string) int {
	switch risk {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func appendUnique(values []string, more ...string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range more {
		if value == "" || seen[value] {
			continue
		}
		values = append(values, value)
		seen[value] = true
	}
	return values
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func mapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if typed, ok := item.(map[string]any); ok {
			out = append(out, typed)
		}
	}
	return out
}

func mapFromPath(value map[string]any, keys ...string) map[string]any {
	current := value
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func stringSliceFromPath(value map[string]any, keys ...string) []string {
	return stringSlice(mapFromPath(value, keys[:len(keys)-1]...)[keys[len(keys)-1]])
}

func stringSliceFromMap(value map[string]any, key string) []string {
	if value == nil {
		return nil
	}
	return stringSlice(value[key])
}

func stringFromMap(value map[string]any, key string) string {
	if value == nil {
		return ""
	}
	out, _ := value[key].(string)
	return out
}

func boolFromMap(value map[string]any, key string) bool {
	if value == nil {
		return false
	}
	out, _ := value[key].(bool)
	return out
}
