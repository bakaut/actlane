# Actlane Is Not An Agent Framework

An agent framework decides what to do next.

Actlane defines safe lanes for what an agent is allowed to do.

## Difference

```text
Agent framework = planning, memory, tool selection, execution loop
Actlane = capability contract, generated artifacts, policy bundle
```

Actlane should work with existing agents and runtimes:

```text
Codex
OpenCode
VS Code
Cursor
Cline
Continue
Custom GPT
Orchestra
```

## Why This Matters

Building an agent framework would pull Actlane away from its core value:

```text
portable policy-aware capability packs
```

The first product surface should be the pack and generated artifacts, not a new runtime.

## Correct Boundary

Actlane may generate instructions, tool metadata, OpenAPI Actions, MCP metadata, IDE snippets, and policy bundles.

The agent runtime still decides when to act.
Actlane defines what a safe action contract looks like.
