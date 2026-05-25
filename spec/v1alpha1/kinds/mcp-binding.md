# MCPBinding

`MCPBinding` describes real MCP server and tool wiring for an Actlane capability.

The contract owns:

- capability reference;
- MCP server name, provider, transport, command, args, env, URL, headers, OAuth, timeout, and enabled state;
- required tool names, server names, toolsets, and required scopes;
- optional generated Actlane policy gate tools.

It does not own risk classification, approval boundaries, forbidden paths, evidence rules, or human validation policy. Those belong to `ResponsibilityContract` and `ToolCallPolicy`.
