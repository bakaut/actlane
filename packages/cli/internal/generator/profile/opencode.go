package profile

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type openCodeRenderer struct{}

func (openCodeRenderer) Render(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) error {
	if err := renderOpenCodeConfig(files, loaded, targetProfile); err != nil {
		return err
	}
	return renderOpenCodeProfileFiles(files, loaded, capability, targetProfile)
}

func renderOpenCodeConfig(files map[string][]byte, loaded *pack.LoadedPack, targetProfile pack.TargetProfile) error {
	if !targetProfile.Spec.Generate.Config {
		return nil
	}
	configPath, err := targetConfigPath(targetProfile)
	if err != nil {
		return err
	}
	files[configPath] = mustJSON(openCodeConfig(loaded, targetProfile))
	return nil
}

func openCodeConfig(loaded *pack.LoadedPack, targetProfile pack.TargetProfile) map[string]any {
	opencode := targetProfile.Spec.OpenCode.Config
	config := map[string]any{}
	if opencode.Schema != "" {
		config["$schema"] = opencode.Schema
	}
	if len(opencode.Instructions) > 0 {
		config["instructions"] = opencode.Instructions
	}
	if len(opencode.Permission) > 0 {
		config["permission"] = opencode.Permission
	}
	if targetProfile.Spec.Generate.MCP && len(loaded.MCPBindings) > 0 {
		config["mcp"] = openCodeMCPServerConfig(loaded.MCPBindings)
	} else if opencode.MCP.ServerName != "" {
		server := map[string]any{}
		if opencode.MCP.Type != "" {
			server["type"] = opencode.MCP.Type
		}
		if len(opencode.MCP.Command) > 0 {
			server["command"] = opencode.MCP.Command
		}
		config["mcp"] = map[string]any{opencode.MCP.ServerName: server}
	}
	return config
}

func openCodeMCPServerConfig(bindings []pack.MCPBinding) map[string]any {
	servers := map[string]any{}
	for _, binding := range bindings {
		for _, server := range binding.Spec.Servers {
			config := map[string]any{
				"enabled": true,
				"type":    "local",
			}
			command := append([]string{}, server.Command...)
			command = append(command, server.Args...)
			if len(command) > 0 {
				config["command"] = command
			}
			if len(server.Env) > 0 {
				config["environment"] = openCodeEnvironment(server.Env)
			}
			servers[server.Name] = config
		}
	}
	return servers
}

func openCodeEnvironment(env map[string]any) map[string]any {
	converted := map[string]any{}
	for key, value := range env {
		if fromEnv, ok := fromEnvRef(value); ok {
			converted[key] = "{env:" + fromEnv + "}"
			continue
		}
		converted[key] = value
	}
	return converted
}

func fromEnvRef(value any) (string, bool) {
	ref, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	fromEnv, ok := ref["fromEnv"].(string)
	return fromEnv, ok && fromEnv != ""
}

func renderOpenCodeProfileFiles(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) error {
	if !capability.Spec.Projections.OpenCode.Enabled {
		return nil
	}
	for _, file := range targetProfile.Spec.OpenCode.Files {
		if !openCodeGeneratedFileEnabled(file.TargetPath, targetProfile.Spec.Generate) {
			continue
		}
		content, ok := openCodeGeneratedFileContent(file.TargetPath, loaded, capability, targetProfile)
		if !ok {
			continue
		}
		rel, err := targetProfileGeneratedPath(targetProfile, file)
		if err != nil {
			return fmt.Errorf("invalid generated OpenCode file path %q: %w", file.GeneratedPath, err)
		}
		if _, exists := files[rel]; exists {
			return fmt.Errorf("duplicate generated OpenCode file path %q", rel)
		}
		files[rel] = []byte(content)
	}
	return nil
}

func openCodeGeneratedFileEnabled(targetPath string, generate pack.TargetProfileGenerate) bool {
	cleaned := path.Clean(filepath.ToSlash(strings.TrimSpace(targetPath)))
	switch {
	case strings.HasPrefix(cleaned, ".opencode/commands/"):
		return generate.Commands
	case strings.HasPrefix(cleaned, ".opencode/agents/"):
		return generate.Agents
	case strings.HasPrefix(cleaned, ".opencode/skills/"):
		return generate.Skills
	default:
		return false
	}
}

func openCodeGeneratedFileContent(targetPath string, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) (string, bool) {
	cleaned := path.Clean(filepath.ToSlash(strings.TrimSpace(targetPath)))
	switch {
	case strings.HasPrefix(cleaned, ".opencode/commands/"):
		return renderOpenCodeCommand(capability, loaded.MCPBindings), true
	case strings.HasPrefix(cleaned, ".opencode/agents/"):
		return renderOpenCodeAgent(capability, targetProfile, loaded.MCPBindings), true
	case strings.HasPrefix(cleaned, ".opencode/skills/") && path.Base(cleaned) == "SKILL.md":
		return renderOpenCodeSkill(capability, loaded.MCPBindings), true
	default:
		return "", false
	}
}

func renderOpenCodeCommand(capability pack.Capability, bindings []pack.MCPBinding) string {
	agent := capability.Spec.Projections.OpenCode.Agent
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: " + yamlScalar(openCodeDescription(capability)) + "\n")
	if agent != "" {
		b.WriteString("agent: " + yamlScalar(agent) + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString("Run capability `" + capability.Metadata.Name + "` for $ARGUMENTS.\n\n")
	writeGateFlow(&b)
	writeCapabilityUse(&b, capability, bindings)
	return ensureTrailingNewline(b.String())
}

func renderOpenCodeAgent(capability pack.Capability, targetProfile pack.TargetProfile, bindings []pack.MCPBinding) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: " + yamlScalar(openCodeDescription(capability)) + "\n")
	b.WriteString("mode: primary\n")
	if len(targetProfile.Spec.OpenCode.Config.Permission) > 0 {
		b.WriteString("permission:\n")
		for _, key := range sortedKeys(targetProfile.Spec.OpenCode.Config.Permission) {
			b.WriteString("  " + key + ": " + yamlScalar(targetProfile.Spec.OpenCode.Config.Permission[key]) + "\n")
		}
	}
	b.WriteString("---\n\n")
	b.WriteString("Use the Actlane capability `" + capability.Metadata.Name + "`.\n\n")
	b.WriteString("Operational rules:\n\n")
	b.WriteString("- Inspect the current project before mutation.\n")
	b.WriteString("- Record marker `actlane:inspect:" + capability.Metadata.Name + "` in working notes.\n")
	b.WriteString("- Require explicit user confirmation before mutating tools.\n")
	b.WriteString("- Stop when policy denies the request or required MCP tools are unavailable.\n\n")
	writeGateFlow(&b)
	writeCapabilityUse(&b, capability, bindings)
	return ensureTrailingNewline(b.String())
}

func renderOpenCodeSkill(capability pack.Capability, bindings []pack.MCPBinding) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + yamlScalar(capability.Metadata.Name) + "\n")
	b.WriteString("description: " + yamlScalar(openCodeDescription(capability)) + "\n")
	b.WriteString("compatibility: opencode\n")
	b.WriteString("metadata:\n")
	b.WriteString("  capability: " + yamlScalar(capability.Metadata.Name) + "\n")
	if capability.Spec.ExecutionRef.Name != "" {
		b.WriteString("  execution_ref: " + yamlScalar(capability.Spec.ExecutionRef.Name) + "\n")
	}
	b.WriteString("---\n\n")
	writeGateFlow(&b)
	writeCapabilityUse(&b, capability, bindings)
	return ensureTrailingNewline(b.String())
}

func writeGateFlow(b *strings.Builder) {
	b.WriteString("Security gate flow:\n\n")
	b.WriteString("- Call `create_github_draft_pr_enforce` before any downstream GitHub MCP tool.\n")
	b.WriteString("- If the gate returns `allowed: false` or `isError: true`, stop and report the policy reasons.\n")
	b.WriteString("- If the gate returns `allowed: true`, use `mutatedInput` and the returned `next` calls for downstream MCP execution.\n")
	b.WriteString("- Do not use original input for downstream MCP calls after mutation.\n\n")
}

func writeCapabilityUse(b *strings.Builder, capability pack.Capability, bindings []pack.MCPBinding) {
	if capability.Metadata.Description != "" {
		b.WriteString(capability.Metadata.Description + "\n\n")
	}
	writeStringList(b, "When to use", capability.Spec.Intent.WhenToUse)
	writeStringList(b, "When not to use", capability.Spec.Intent.WhenNotToUse)
	if len(capability.Spec.WorkflowHints) > 0 {
		b.WriteString("Workflow:\n\n")
		for _, hint := range capability.Spec.WorkflowHints {
			b.WriteString("- `" + hint.Step + "`: " + hint.Purpose + "\n")
		}
		b.WriteString("\n")
	}
	requiredInputs := requiredInputFields(capability)
	writeStringList(b, "Required inputs", requiredInputs)
	tools := openCodeToolNames(bindings, capability)
	writeStringList(b, "MCP tools", tools)
}

func writeStringList(b *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString(title + ":\n\n")
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		b.WriteString("- " + value + "\n")
	}
	b.WriteString("\n")
}

func requiredInputFields(capability pack.Capability) []string {
	input, ok := capability.Spec.Interface.Input["required"].([]any)
	if !ok {
		return nil
	}
	fields := make([]string, 0, len(input))
	for _, value := range input {
		name, ok := value.(string)
		if ok && name != "" {
			fields = append(fields, "`"+name+"`")
		}
	}
	return fields
}

func openCodeToolNames(bindings []pack.MCPBinding, capability pack.Capability) []string {
	var names []string
	for _, binding := range bindings {
		if binding.Spec.CapabilityRef.Name != capability.Metadata.Name {
			continue
		}
		for _, tool := range generatedTools(binding) {
			if tool.Name != "" {
				names = append(names, "`"+tool.Name+"`")
			}
		}
		for _, tool := range binding.Spec.RequiredTools {
			if tool.Server == "" || tool.Name == "" {
				continue
			}
			names = append(names, "`"+tool.Server+"_"+tool.Name+"`")
		}
	}
	sort.Strings(names)
	return names
}

func openCodeDescription(capability pack.Capability) string {
	if capability.Metadata.Description != "" {
		return capability.Metadata.Description
	}
	if capability.Metadata.Title != "" {
		return capability.Metadata.Title
	}
	return capability.Metadata.Name
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func yamlScalar(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func ensureTrailingNewline(value string) string {
	return strings.TrimRight(value, "\n") + "\n"
}
