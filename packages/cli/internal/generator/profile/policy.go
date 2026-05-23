package profile

import "github.com/actlane/actlane/packages/cli/internal/pack"

type policyBundle struct {
	Pack         string                   `json:"pack"`
	Version      string                   `json:"version"`
	Target       string                   `json:"target"`
	Capabilities []string                 `json:"capabilities"`
	Decisions    []string                 `json:"decisions"`
	Rules        map[string]any           `json:"rules"`
	MCPBindings  []policyBundleMCPBinding `json:"mcpBindings"`
}

type policyBundleMCPBinding struct {
	Name           string                  `json:"name"`
	Capability     string                  `json:"capability"`
	Handler        string                  `json:"handler"`
	GeneratedTools []pack.MCPGeneratedTool `json:"generatedTools,omitempty"`
	RequiredTools  []pack.MCPToolBinding   `json:"requiredTools,omitempty"`
}

func collectRules(policies []pack.Policy) map[string]any {
	rules := map[string]any{}
	for _, policy := range policies {
		if len(policy.Spec.Match.Capabilities) > 0 {
			rules["capabilities"] = policy.Spec.Match.Capabilities
		}
		if len(policy.Spec.Mutate.Defaults) > 0 {
			rules["defaults"] = policy.Spec.Mutate.Defaults
		}
		if policy.Spec.Mutate.Ensure.BranchPrefix != "" {
			rules["branchPrefix"] = policy.Spec.Mutate.Ensure.BranchPrefix
		}
		if policy.Spec.Validate.Confirmation.Field != "" {
			rules["confirmation"] = policy.Spec.Validate.Confirmation
		}
		if len(policy.Spec.Validate.RepoAllowlist) > 0 {
			rules["repoAllowlist"] = policy.Spec.Validate.RepoAllowlist
		}
		if len(policy.Spec.Validate.ForbidPaths) > 0 {
			rules["forbidPaths"] = policy.Spec.Validate.ForbidPaths
		}
		if policy.Spec.Validate.Limits.MaxFiles != 0 {
			rules["maxFiles"] = policy.Spec.Validate.Limits.MaxFiles
		}
		if policy.Spec.Validate.Limits.MaxDiffKB != 0 {
			rules["maxDiffKb"] = policy.Spec.Validate.Limits.MaxDiffKB
		}
		if policy.Spec.Approval.Required {
			rules["approval"] = policy.Spec.Approval
		}
		if policy.Spec.Audit.Level != "" || len(policy.Spec.Audit.Include) > 0 {
			rules["audit"] = policy.Spec.Audit
		}
	}
	return rules
}

func policyBundleMCPBindings(bindings []pack.MCPBinding) []policyBundleMCPBinding {
	out := make([]policyBundleMCPBinding, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, policyBundleMCPBinding{
			Name:           binding.Metadata.Name,
			Capability:     binding.Spec.CapabilityRef.Name,
			Handler:        binding.Spec.Strategy.Handler,
			GeneratedTools: generatedTools(binding),
			RequiredTools:  binding.Spec.RequiredTools,
		})
	}
	return out
}
