package pack

import (
	"fmt"
	"regexp"
)

var kebabName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

func Validate(loaded *LoadedPack) error {
	if loaded.Manifest.APIVersion != "actlane.ru/v1alpha1" {
		return fmt.Errorf("CapabilityPack apiVersion must be actlane.ru/v1alpha1")
	}
	if loaded.Manifest.Kind != "CapabilityPack" {
		return fmt.Errorf("actlane.yaml kind must be CapabilityPack")
	}
	if loaded.Manifest.Metadata.Name == "" {
		return fmt.Errorf("CapabilityPack metadata.name is required")
	}
	if len(loaded.Manifest.Spec.Capabilities) == 0 {
		return fmt.Errorf("CapabilityPack spec.capabilities is required")
	}
	if len(loaded.Manifest.Spec.TargetProfiles) == 0 {
		return fmt.Errorf("CapabilityPack spec.targetProfiles is required")
	}
	targetProfiles := map[string]bool{}
	for _, targetProfile := range loaded.TargetProfiles {
		if targetProfile.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("target profile %s apiVersion must be actlane.ru/v1alpha1", targetProfile.Metadata.Name)
		}
		if targetProfile.Kind != "TargetProfile" {
			return fmt.Errorf("target profile %s kind must be TargetProfile", targetProfile.Metadata.Name)
		}
		if targetProfile.Metadata.Name == "" {
			return fmt.Errorf("target profile metadata.name is required")
		}
		if targetProfile.Spec.Target == "" {
			return fmt.Errorf("target profile %s spec.target is required", targetProfile.Metadata.Name)
		}
		if targetProfile.Spec.Output.Root == "" {
			return fmt.Errorf("target profile %s spec.output.root is required", targetProfile.Metadata.Name)
		}
		if targetProfile.Spec.Output.Config == "" && targetProfile.Spec.OpenCode.Config.Filename == "" {
			return fmt.Errorf("target profile %s spec.output.config is required", targetProfile.Metadata.Name)
		}
		targetProfiles[targetProfile.Spec.Target] = true
	}
	for _, target := range loaded.Manifest.Spec.Targets {
		if !targetProfiles[target] {
			return fmt.Errorf("target %q has no target profile", target)
		}
	}

	policies := map[string]bool{}
	for _, policy := range loaded.Policies {
		if policy.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("policy %s apiVersion must be actlane.ru/v1alpha1", policy.Metadata.Name)
		}
		if policy.Kind != "ToolCallPolicy" {
			return fmt.Errorf("policy %s kind must be ToolCallPolicy", policy.Metadata.Name)
		}
		if policy.Metadata.Name == "" {
			return fmt.Errorf("policy metadata.name is required")
		}
		policies[policy.Metadata.Name] = true
	}

	bindings := map[string]bool{}
	for _, binding := range loaded.MCPBindings {
		if binding.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("mcp binding %s apiVersion must be actlane.ru/v1alpha1", binding.Metadata.Name)
		}
		if binding.Kind != "MCPBinding" {
			return fmt.Errorf("mcp binding %s kind must be MCPBinding", binding.Metadata.Name)
		}
		if binding.Metadata.Name == "" {
			return fmt.Errorf("mcp binding metadata.name is required")
		}
		bindings[binding.Metadata.Name] = true
	}

	for _, capability := range loaded.Capabilities {
		if capability.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("capability %s apiVersion must be actlane.ru/v1alpha1", capability.Metadata.Name)
		}
		if capability.Kind != "Capability" {
			return fmt.Errorf("capability %s kind must be Capability", capability.Metadata.Name)
		}
		if !kebabName.MatchString(capability.Metadata.Name) {
			return fmt.Errorf("capability metadata.name must be kebab-case")
		}
		if capability.Spec.WhenToUse == "" && len(capability.Spec.Intent.WhenToUse) == 0 {
			return fmt.Errorf("capability %s usage guidance is required", capability.Metadata.Name)
		}
		if len(capabilityTargets(capability)) == 0 {
			return fmt.Errorf("capability %s spec.targets is required", capability.Metadata.Name)
		}
		for _, target := range capabilityTargets(capability) {
			if !targetProfiles[target] {
				return fmt.Errorf("capability %s target %q has no target profile", capability.Metadata.Name, target)
			}
			if _, ok := capability.Spec.Profiles[target]; !ok && !capability.Spec.Projections.OpenCode.Enabled {
				return fmt.Errorf("capability %s missing spec.profiles.%s", capability.Metadata.Name, target)
			}
		}
		if len(capability.Spec.Inputs) == 0 && len(capability.Spec.Interface.Input) == 0 {
			return fmt.Errorf("capability %s spec.inputs is required", capability.Metadata.Name)
		}
		if len(capability.Spec.Outputs) == 0 && len(capability.Spec.Interface.Output) == 0 {
			return fmt.Errorf("capability %s spec.outputs is required", capability.Metadata.Name)
		}
		if len(capability.Spec.Policies) == 0 && capability.Spec.PolicyRef.Name == "" {
			return fmt.Errorf("capability %s spec.policies is required", capability.Metadata.Name)
		}
		for _, policyName := range capabilityPolicies(capability) {
			if !policies[policyName] {
				return fmt.Errorf("capability %s references missing policy %s", capability.Metadata.Name, policyName)
			}
		}
		if capability.Spec.ExecutionRef.Name != "" && !bindings[capability.Spec.ExecutionRef.Name] {
			return fmt.Errorf("capability %s references missing mcp binding %s", capability.Metadata.Name, capability.Spec.ExecutionRef.Name)
		}
	}

	return nil
}

func capabilityTargets(capability Capability) []string {
	if len(capability.Spec.Targets) > 0 {
		return capability.Spec.Targets
	}
	targets := []string{}
	if capability.Spec.Projections.OpenCode.Enabled {
		targets = append(targets, "opencode")
	}
	return targets
}

func capabilityPolicies(capability Capability) []string {
	if len(capability.Spec.Policies) > 0 {
		return capability.Spec.Policies
	}
	if capability.Spec.PolicyRef.Name != "" {
		return []string{capability.Spec.PolicyRef.Name}
	}
	return nil
}
