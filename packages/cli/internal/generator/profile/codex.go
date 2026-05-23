package profile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type codexRenderer struct{}

func (codexRenderer) Render(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) error {
	if err := renderCodexConfig(files, loaded, targetProfile); err != nil {
		return err
	}
	return nil
}

func renderCodexConfig(files map[string][]byte, loaded *pack.LoadedPack, targetProfile pack.TargetProfile) error {
	if !targetProfile.Spec.Generate.Config {
		return nil
	}
	configPath, err := targetConfigPath(targetProfile)
	if err != nil {
		return err
	}
	files[configPath] = []byte(renderCodexMCPConfig(loaded.MCPBindings))
	return nil
}

func renderCodexMCPConfig(bindings []pack.MCPBinding) string {
	var b strings.Builder
	b.WriteString("# Generated Actlane MCP config snippet for Codex.\n")
	b.WriteString("# Merge these tables into ~/.codex/config.toml when Codex does not load project-local config.\n\n")
	for _, server := range sortedMCPServers(bindings) {
		command := append([]string{}, server.Command...)
		command = append(command, server.Args...)
		if len(command) == 0 {
			continue
		}
		b.WriteString("[mcp_servers." + server.Name + "]\n")
		b.WriteString("command = " + tomlString(command[0]) + "\n")
		if len(command) > 1 {
			b.WriteString("args = " + tomlStringArray(command[1:]) + "\n")
		}
		if len(server.Env) > 0 {
			b.WriteString("\n[mcp_servers." + server.Name + ".env]\n")
			for _, key := range sortedAnyKeys(server.Env) {
				b.WriteString(key + " = " + tomlString(codexEnvValue(server.Env[key])) + "\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func sortedMCPServers(bindings []pack.MCPBinding) []pack.MCPRuntimeServer {
	var servers []pack.MCPRuntimeServer
	for _, binding := range bindings {
		servers = append(servers, binding.Spec.Servers...)
	}
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})
	return servers
}

func sortedAnyKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func codexEnvValue(value any) string {
	if fromEnv, ok := fromEnvRef(value); ok {
		return "${" + fromEnv + "}"
	}
	return fmt.Sprint(value)
}

func tomlString(value string) string {
	escaped := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	).Replace(value)
	return "\"" + escaped + "\""
}

func tomlStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, tomlString(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
