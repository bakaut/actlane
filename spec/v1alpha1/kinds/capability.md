# Capability

`Capability` describes one high-level agent action.

It should expose a user-meaningful workflow instead of raw tool calls.

Example:

```text
create-safe-draft-pr
```

## Required Ideas

- when the capability should be used;
- input schema;
- output schema;
- policy references;
- execution binding reference;
- optional responsibility contract reference;
- reporting contract for the agent.

## Boundary

`Capability` must not own target-specific file layout, generated paths, prompts for a specific runtime, or exact MCP server/tool wiring.

Use `TargetProfile` for generated file layout, `SkillContract` for skill content, `CommandContract` for command entrypoints, `AgentContract` for role boundaries, `MCPBinding` for real tools, and `ToolCallPolicy` for enforcement.
