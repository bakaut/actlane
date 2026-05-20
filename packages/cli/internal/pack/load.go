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

	manifestPath := filepath.Join(abs, "actlane.yaml")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read actlane.yaml: %w", err)
	}
	var manifest CapabilityPack
	if err := yaml.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, fmt.Errorf("parse actlane.yaml: %w", err)
	}

	loaded := &LoadedPack{
		Root:         abs,
		ManifestPath: manifestPath,
		Manifest:     manifest,
		ManifestRaw:  manifestRaw,
	}

	for _, rel := range manifest.Spec.Capabilities {
		path := filepath.Join(abs, rel)
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

	for _, rel := range manifest.Spec.Policies {
		path := filepath.Join(abs, rel)
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

	return loaded, nil
}
