# The Problem

AI agents are gaining access to real tools: repositories, CI systems, ticket trackers, memory, internal APIs, file systems, deploy pipelines, and MCP servers.

The rules that make those tools safe are usually scattered across:

```text
prompt
AGENTS.md
SKILL.md
MCP tool schema
CLI/Codex/Claude/OpenCode config
opencode.json
.claude/mcp.json
gateway policy
CI scripts
README
an engineer's memory
```

This creates three practical problems.

## Capability Drift

The same workflow is described differently for each runtime. A "safe draft PR" workflow might exist as a prompt paragraph, an MCP tool schema, a skill, an CLI/Codex/Claude/OpenCode config, and a gateway rule.

When one copy changes, the others silently drift.

## Prompt-Only Safety

Prompts can ask agents to be careful:

```text
do not touch secrets
only create draft PRs
run tests first
use a safe branch prefix
```

But a prompt is not a reproducible contract and not enforcement.

## Unsafe Adoption

Teams need to try agent capabilities inside existing projects, but generated files can be hard to review, update, or remove.

The default must be:

```text
generate separately
show the diff
apply explicitly
remove only owned files and owned blocks
```

## Actlane's Question

How do we safely, portably, and reproducibly describe:

```text
what an agent may do
when it should use an action
which inputs are valid
which defaults should be applied
which outputs should be generated
what must be denied
what must be audited
where the same policy can later be enforced
```
