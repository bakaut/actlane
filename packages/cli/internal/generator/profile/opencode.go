package profile

import (
	"github.com/actlane/actlane/packages/cli/internal/pack"
)

type openCodeRenderer struct{}

func (openCodeRenderer) Render(files map[string][]byte, loaded *pack.LoadedPack, capability pack.Capability, targetProfile pack.TargetProfile) error {
	if err := renderOpenCodeConfig(files, loaded, targetProfile); err != nil {
		return err
	}
	return nil
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
				"type":    defaultMCPType(server.Transport),
			}
			if server.Enabled != nil {
				config["enabled"] = *server.Enabled
			}
			command := append([]string{}, server.Command...)
			command = append(command, server.Args...)
			if len(command) > 0 {
				config["command"] = command
			}
			if server.URL != "" {
				config["url"] = server.URL
			}
			if len(server.Env) > 0 {
				config["environment"] = openCodeEnvironment(server.Env)
			}
			if len(server.Headers) > 0 {
				config["headers"] = openCodeEnvironment(server.Headers)
			}
			if server.OAuth != nil {
				config["oauth"] = server.OAuth
			}
			if server.Timeout > 0 {
				config["timeout"] = server.Timeout
			}
			servers[server.Name] = config
		}
	}
	return servers
}

func defaultMCPType(value string) string {
	if value == "" || value == "stdio" {
		return "local"
	}
	return value
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
