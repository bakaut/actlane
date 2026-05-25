package pack

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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
		if targetProfile.Spec.Output.Config == "" && targetProfile.Spec.Codex.Config.Filename == "" && targetProfile.Spec.OpenCode.Config.Filename == "" {
			return fmt.Errorf("target profile %s spec.output.config is required", targetProfile.Metadata.Name)
		}
		targetProfiles[targetProfile.Spec.Target] = true
	}
	for _, target := range loaded.Manifest.Spec.Targets {
		if !targetProfiles[target] {
			return fmt.Errorf("target %q has no target profile", target)
		}
	}
	capabilities := map[string]bool{}
	for _, capability := range loaded.Capabilities {
		if capability.Metadata.Name != "" {
			capabilities[capability.Metadata.Name] = true
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

	contracts := map[string]bool{}
	for _, contract := range loaded.Contracts {
		if contract.Metadata.Name != "" {
			contracts[contract.Metadata.Name] = true
		}
	}

	skills := map[string]bool{}
	for _, skill := range loaded.Skills {
		if skill.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("skill contract %s apiVersion must be actlane.ru/v1alpha1", skill.Metadata.Name)
		}
		if skill.Kind != "SkillContract" {
			return fmt.Errorf("skill contract %s kind must be SkillContract", skill.Metadata.Name)
		}
		if !kebabName.MatchString(skill.Metadata.Name) {
			return fmt.Errorf("skill contract metadata.name must be kebab-case")
		}
		if skill.Metadata.Description == "" {
			return fmt.Errorf("skill contract %s metadata.description is required", skill.Metadata.Name)
		}
		if skill.Spec.Body == "" {
			return fmt.Errorf("skill contract %s spec.body is required", skill.Metadata.Name)
		}
		if strings.Contains(skill.Spec.Body, "Required inputs:") || strings.Contains(skill.Spec.Body, "MCP tools:") {
			return fmt.Errorf("skill contract %s must not embed generated input or MCP tool sections", skill.Metadata.Name)
		}
		if err := validateSkillResources(skill, "scripts", skill.Spec.Scripts); err != nil {
			return err
		}
		if err := validateSkillResources(skill, "references", skill.Spec.References); err != nil {
			return err
		}
		if err := validateSkillResources(skill, "assets", skill.Spec.Assets); err != nil {
			return err
		}
		skills[skill.Metadata.Name] = true
	}

	commands := map[string]bool{}
	agents := map[string]bool{}
	for _, agent := range loaded.Agents {
		if agent.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("agent contract %s apiVersion must be actlane.ru/v1alpha1", agent.Metadata.Name)
		}
		if agent.Kind != "AgentContract" {
			return fmt.Errorf("agent contract %s kind must be AgentContract", agent.Metadata.Name)
		}
		if !kebabName.MatchString(agent.Metadata.Name) {
			return fmt.Errorf("agent contract metadata.name must be kebab-case")
		}
		if agent.Metadata.Description == "" {
			return fmt.Errorf("agent contract %s metadata.description is required", agent.Metadata.Name)
		}
		if agent.Spec.Mode != "primary" && agent.Spec.Mode != "subagent" {
			return fmt.Errorf("agent contract %s spec.mode must be primary or subagent", agent.Metadata.Name)
		}
		if agent.Spec.Role.Summary == "" {
			return fmt.Errorf("agent contract %s spec.role.summary is required", agent.Metadata.Name)
		}
		if specHasKey(agent.Raw, "permissions") {
			return fmt.Errorf("agent contract %s must not define spec.permissions; use ResponsibilityContract and TargetProfile mapping", agent.Metadata.Name)
		}
		if specHasKey(agent.Raw, "output") {
			return fmt.Errorf("agent contract %s must not define spec.output; use Capability reporting or ResponsibilityContract evidence", agent.Metadata.Name)
		}
		if specHasKey(agent.Raw, "projections") {
			return fmt.Errorf("agent contract %s must not define spec.projections; use TargetProfile files", agent.Metadata.Name)
		}
		for _, capabilityName := range agent.Spec.Capabilities.Allowed {
			if !capabilities[capabilityName] {
				return fmt.Errorf("agent contract %s references missing capability %s", agent.Metadata.Name, capabilityName)
			}
		}
		for _, skillName := range agent.Spec.Skills.Allowed {
			if !skills[skillName] {
				return fmt.Errorf("agent contract %s references missing skill %s", agent.Metadata.Name, skillName)
			}
		}
		if agents[agent.Metadata.Name] {
			return fmt.Errorf("duplicate agent contract %s", agent.Metadata.Name)
		}
		agents[agent.Metadata.Name] = true
	}

	for _, command := range loaded.Commands {
		if command.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("command contract %s apiVersion must be actlane.ru/v1alpha1", command.Metadata.Name)
		}
		if command.Kind != "CommandContract" {
			return fmt.Errorf("command contract %s kind must be CommandContract", command.Metadata.Name)
		}
		if !kebabName.MatchString(command.Metadata.Name) {
			return fmt.Errorf("command contract metadata.name must be kebab-case")
		}
		if command.Metadata.Description == "" {
			return fmt.Errorf("command contract %s metadata.description is required", command.Metadata.Name)
		}
		if command.Spec.CapabilityRef.Name == "" {
			return fmt.Errorf("command contract %s spec.capabilityRef.name is required", command.Metadata.Name)
		}
		if command.Spec.Invocation.Slash == "" {
			return fmt.Errorf("command contract %s spec.invocation.slash is required", command.Metadata.Name)
		}
		if command.Spec.Prompt.Template == "" {
			return fmt.Errorf("command contract %s spec.prompt.template is required", command.Metadata.Name)
		}
		if specHasKey(command.Raw, "safety") {
			return fmt.Errorf("command contract %s must not define spec.safety; use ToolCallPolicy and ResponsibilityContract", command.Metadata.Name)
		}
		if specHasKey(command.Raw, "output") {
			return fmt.Errorf("command contract %s must not define spec.output; use Capability output and ResponsibilityContract evidence", command.Metadata.Name)
		}
		if specHasKey(command.Raw, "projections") {
			return fmt.Errorf("command contract %s must not define spec.projections; use TargetProfile files", command.Metadata.Name)
		}
		if command.Spec.Arguments.Mode == "passthrough" && command.Spec.Arguments.Placeholder == "" {
			return fmt.Errorf("command contract %s passthrough arguments require placeholder", command.Metadata.Name)
		}
		if command.Spec.AgentRef.Name != "" && !agents[command.Spec.AgentRef.Name] {
			return fmt.Errorf("command contract %s references missing agent %s", command.Metadata.Name, command.Spec.AgentRef.Name)
		}
		commands[command.Metadata.Name] = true
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
		if len(capability.Spec.Profiles) > 0 {
			return fmt.Errorf("capability %s must not define spec.profiles; use TargetProfile files", capability.Metadata.Name)
		}
		if len(capability.Spec.MCP.Tools) > 0 {
			return fmt.Errorf("capability %s must not define exact MCP tools; use MCPBinding", capability.Metadata.Name)
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
		if capability.Spec.ResponsibilityRef.Name != "" && !contracts[capability.Spec.ResponsibilityRef.Name] {
			return fmt.Errorf("capability %s references missing responsibility contract %s", capability.Metadata.Name, capability.Spec.ResponsibilityRef.Name)
		}
	}

	for _, contract := range loaded.Contracts {
		if contract.APIVersion != "actlane.ru/v1alpha1" {
			return fmt.Errorf("responsibility contract %s apiVersion must be actlane.ru/v1alpha1", contract.Metadata.Name)
		}
		if contract.Kind != "ResponsibilityContract" {
			return fmt.Errorf("responsibility contract %s kind must be ResponsibilityContract", contract.Metadata.Name)
		}
		if contract.Metadata.Name == "" {
			return fmt.Errorf("responsibility contract metadata.name is required")
		}
		if specHasKey(contract.Raw, "ci") {
			return fmt.Errorf("responsibility contract %s must not define spec.ci; keep CI implementation outside the responsibility contract", contract.Metadata.Name)
		}
		if specHasKey(contract.Raw, "acceptanceCriteria") {
			return fmt.Errorf("responsibility contract %s must not define spec.acceptanceCriteria; use docs or tests", contract.Metadata.Name)
		}
		if responsibilityHasExactMCPTools(contract.Raw) {
			return fmt.Errorf("responsibility contract %s must not define exact MCP tools; use MCPBinding", contract.Metadata.Name)
		}
	}

	for _, targetProfile := range loaded.TargetProfiles {
		for _, file := range targetProfileFiles(targetProfile) {
			if file.SkillContract != "" && !skills[file.SkillContract] {
				return fmt.Errorf("target profile %s references missing skill contract %s", targetProfile.Metadata.Name, file.SkillContract)
			}
			if file.CommandContract != "" && !commands[file.CommandContract] {
				return fmt.Errorf("target profile %s references missing command contract %s", targetProfile.Metadata.Name, file.CommandContract)
			}
			if file.AgentContract != "" && !agents[file.AgentContract] {
				return fmt.Errorf("target profile %s references missing agent contract %s", targetProfile.Metadata.Name, file.AgentContract)
			}
			if countTargetFileGenerators(file) > 1 {
				return fmt.Errorf("target profile %s file %q must use at most one generator", targetProfile.Metadata.Name, file.TargetPath)
			}
		}
	}

	for _, command := range loaded.Commands {
		found := false
		for _, capability := range loaded.Capabilities {
			if capability.Metadata.Name == command.Spec.CapabilityRef.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("command contract %s references missing capability %s", command.Metadata.Name, command.Spec.CapabilityRef.Name)
		}
	}

	return nil
}

func specHasKey(raw []byte, key string) bool {
	spec, ok := rawSpec(raw)
	if !ok {
		return false
	}
	_, exists := spec[key]
	return exists
}

func rawSpec(raw []byte) (map[string]any, bool) {
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false
	}
	spec, ok := doc["spec"].(map[string]any)
	return spec, ok
}

func responsibilityHasExactMCPTools(raw []byte) bool {
	spec, ok := rawSpec(raw)
	if !ok {
		return false
	}
	tools, ok := spec["tools"].(map[string]any)
	if !ok {
		return false
	}
	mcp, ok := tools["mcp"].(map[string]any)
	if !ok {
		return false
	}
	if _, ok := mcp["servers"]; ok {
		return true
	}
	return containsKeyDeep(mcp, "allowedTools") || containsKeyDeep(mcp, "deniedTools")
}

func containsKeyDeep(value any, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			if k == key || containsKeyDeep(v, key) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if containsKeyDeep(item, key) {
				return true
			}
		}
	}
	return false
}

func validateSkillResources(skill SkillContract, group string, resources []SkillResource) error {
	for _, resource := range resources {
		if resource.Source == "" || resource.Path == "" {
			return fmt.Errorf("skill contract %s %s resource must declare source and path", skill.Metadata.Name, group)
		}
		cleaned := path.Clean(resource.Path)
		if cleaned == "." || path.IsAbs(cleaned) || strings.HasPrefix(cleaned, "../") {
			return fmt.Errorf("skill contract %s %s resource path %q must be relative", skill.Metadata.Name, group, resource.Path)
		}
		if !strings.HasPrefix(cleaned, group+"/") {
			return fmt.Errorf("skill contract %s %s resource path %q must be under %s/", skill.Metadata.Name, group, resource.Path, group)
		}
	}
	return nil
}

func capabilityTargets(capability Capability) []string {
	if len(capability.Spec.Targets) > 0 {
		return capability.Spec.Targets
	}
	targets := []string{}
	if capability.Spec.Projections.Codex.Enabled {
		targets = append(targets, "codex")
	}
	if capability.Spec.Projections.OpenCode.Enabled {
		targets = append(targets, "opencode")
	}
	return targets
}

func capabilityProjectionEnabled(capability Capability, target string) bool {
	switch target {
	case "codex":
		return capability.Spec.Projections.Codex.Enabled
	case "opencode":
		return capability.Spec.Projections.OpenCode.Enabled
	default:
		return false
	}
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

func targetProfileFiles(targetProfile TargetProfile) []TargetProfileFile {
	switch targetProfile.Spec.Target {
	case "codex":
		return targetProfile.Spec.Codex.Files
	case "opencode":
		return targetProfile.Spec.OpenCode.Files
	default:
		return nil
	}
}

func countTargetFileGenerators(file TargetProfileFile) int {
	count := 0
	if file.Source != "" {
		count++
	}
	if file.SkillContract != "" {
		count++
	}
	if file.CommandContract != "" {
		count++
	}
	if file.AgentContract != "" {
		count++
	}
	return count
}
