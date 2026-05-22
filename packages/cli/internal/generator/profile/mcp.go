package profile

import (
	"fmt"
	"sort"

	"github.com/actlane/actlane/packages/cli/internal/pack"
)

func renderMCPBindingArtifacts(files map[string][]byte, loaded *pack.LoadedPack) {
	if len(loaded.MCPBindings) == 0 {
		return
	}
	files["generated/mcp/server.json"] = mustJSON(map[string]any{
		"mcp": mcpServerConfig(loaded.MCPBindings),
	})
	files["generated/mcp/tools.json"] = mustJSON(map[string]any{
		"bindings": mcpToolBindings(loaded.MCPBindings),
	})
}

func mcpServerConfig(bindings []pack.MCPBinding) map[string]any {
	servers := map[string]any{}
	for _, binding := range bindings {
		for _, server := range binding.Spec.Servers {
			config := map[string]any{
				"provider":  server.Provider,
				"source":    server.Source,
				"transport": server.Transport,
			}
			command := append([]string{}, server.Command...)
			command = append(command, server.Args...)
			if len(command) > 0 {
				config["command"] = command
			}
			if len(server.Env) > 0 {
				config["environment"] = server.Env
			}
			servers[server.Name] = config
		}
	}
	return servers
}

func mcpToolBindings(bindings []pack.MCPBinding) []map[string]any {
	var tools []map[string]any
	for _, binding := range bindings {
		for _, tool := range binding.Spec.RequiredTools {
			tools = append(tools, map[string]any{
				"binding":        binding.Metadata.Name,
				"capability":     binding.Spec.CapabilityRef.Name,
				"generatedTool":  binding.Spec.GeneratedTool.Name,
				"name":           tool.Name,
				"server":         tool.Server,
				"toolset":        tool.Toolset,
				"requiredScopes": tool.RequiredScopes,
			})
		}
	}
	sort.Slice(tools, func(i, j int) bool {
		return fmt.Sprint(tools[i]["name"]) < fmt.Sprint(tools[j]["name"])
	})
	return tools
}
