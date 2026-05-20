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
	for _, target := range loaded.Manifest.Spec.Targets {
		if target != "opencode" {
			return fmt.Errorf("unsupported target %q; supported target: opencode", target)
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
		if capability.Spec.WhenToUse == "" {
			return fmt.Errorf("capability %s spec.whenToUse is required", capability.Metadata.Name)
		}
		if len(capability.Spec.Targets) == 0 {
			return fmt.Errorf("capability %s spec.targets is required", capability.Metadata.Name)
		}
		for _, target := range capability.Spec.Targets {
			if target != "opencode" {
				return fmt.Errorf("capability %s unsupported target %q; supported target: opencode", capability.Metadata.Name, target)
			}
		}
		if len(capability.Spec.Inputs) == 0 {
			return fmt.Errorf("capability %s spec.inputs is required", capability.Metadata.Name)
		}
		if len(capability.Spec.Outputs) == 0 {
			return fmt.Errorf("capability %s spec.outputs is required", capability.Metadata.Name)
		}
		if len(capability.Spec.Policies) == 0 {
			return fmt.Errorf("capability %s spec.policies is required", capability.Metadata.Name)
		}
		for _, policyName := range capability.Spec.Policies {
			if !policies[policyName] {
				return fmt.Errorf("capability %s references missing policy %s", capability.Metadata.Name, policyName)
			}
		}
	}

	return nil
}
