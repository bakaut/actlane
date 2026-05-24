package schema

import (
	"fmt"
	"os"
	"path/filepath"
)

var schemaFiles = map[string]string{
	"capability":       "capability.schema.json",
	"capability-pack":  "capability-pack.schema.json",
	"command-contract": "command-contract.schema.json",
	"mcp-binding":      "mcp-binding.schema.json",
	"skill-contract":   "skill-contract.schema.json",
	"target-profile":   "target-profile.schema.json",
	"tool-call-policy": "tool-call-policy.schema.json",
}

func Read(name string) (string, error) {
	file, ok := schemaFiles[name]
	if !ok {
		return "", fmt.Errorf("unknown schema %q", name)
	}
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, "spec/v1alpha1/schemas", file))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "spec/v1alpha1/schemas")); err == nil {
			return wd, nil
		}
		next := filepath.Dir(wd)
		if next == wd {
			return "", fmt.Errorf("repository root with spec/v1alpha1/schemas not found")
		}
		wd = next
	}
}
