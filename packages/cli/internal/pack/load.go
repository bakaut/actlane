package pack

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func Load(root string) (*LoadedPack, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	packRoot := abs
	manifestPath := filepath.Join(packRoot, "actlane.yaml")
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		packRoot = filepath.Dir(abs)
		manifestPath = abs
	}

	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read actlane.yaml: %w", err)
	}
	var manifest CapabilityPack
	if err := yaml.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, fmt.Errorf("parse actlane.yaml: %w", err)
	}

	loaded := &LoadedPack{
		Root:         packRoot,
		ManifestPath: manifestPath,
		Manifest:     manifest,
		ManifestRaw:  manifestRaw,
	}

	for _, rel := range manifest.Spec.Capabilities {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read capability %s: %w", rel, err)
		}
		var capability Capability
		if err := yaml.Unmarshal(raw, &capability); err != nil {
			return nil, fmt.Errorf("parse capability %s: %w", rel, err)
		}
		capability.Path = path
		capability.Raw = raw
		loaded.Capabilities = append(loaded.Capabilities, capability)
	}

	for _, rel := range manifest.Spec.Skills {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read skill contract %s: %w", rel, err)
		}
		var skill SkillContract
		if err := yaml.Unmarshal(raw, &skill); err != nil {
			return nil, fmt.Errorf("parse skill contract %s: %w", rel, err)
		}
		skill.Path = path
		skill.Raw = raw
		loaded.Skills = append(loaded.Skills, skill)
	}

	for _, rel := range manifest.Spec.Commands {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read command contract %s: %w", rel, err)
		}
		var command CommandContract
		if err := yaml.Unmarshal(raw, &command); err != nil {
			return nil, fmt.Errorf("parse command contract %s: %w", rel, err)
		}
		command.Path = path
		command.Raw = raw
		loaded.Commands = append(loaded.Commands, command)
	}

	for _, rel := range manifest.Spec.Agents {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read agent contract %s: %w", rel, err)
		}
		var agent AgentContract
		if err := yaml.Unmarshal(raw, &agent); err != nil {
			return nil, fmt.Errorf("parse agent contract %s: %w", rel, err)
		}
		agent.Path = path
		agent.Raw = raw
		loaded.Agents = append(loaded.Agents, agent)
	}

	for _, rel := range manifest.Spec.Policies {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read policy %s: %w", rel, err)
		}
		var policy Policy
		if err := yaml.Unmarshal(raw, &policy); err != nil {
			return nil, fmt.Errorf("parse policy %s: %w", rel, err)
		}
		policy.Path = path
		policy.Raw = raw
		loaded.Policies = append(loaded.Policies, policy)
	}

	for _, rel := range manifest.Spec.MCPBindings {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read mcp binding %s: %w", rel, err)
		}
		var binding MCPBinding
		if err := yaml.Unmarshal(raw, &binding); err != nil {
			return nil, fmt.Errorf("parse mcp binding %s: %w", rel, err)
		}
		binding.Path = path
		binding.Raw = raw
		loaded.MCPBindings = append(loaded.MCPBindings, binding)
	}

	for _, rel := range manifest.Spec.TargetProfiles {
		path := filepath.Join(packRoot, rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read target profile %s: %w", rel, err)
		}
		var targetProfile TargetProfile
		if err := yaml.Unmarshal(raw, &targetProfile); err != nil {
			return nil, fmt.Errorf("parse target profile %s: %w", rel, err)
		}
		targetProfile.Path = path
		targetProfile.Raw = raw
		loaded.TargetProfiles = append(loaded.TargetProfiles, targetProfile)
	}

	return loaded, nil
}
