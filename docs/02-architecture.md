# Architecture

Actlane is generation-first.

The MVP architecture is:

```text
Source manifests
  -> validator
  -> generator
  -> generated artifacts
  -> optional lockfile
  -> optional runtime enforcement later
```

## Main Components

### Capability

Describes a safe high-level action:

```text
name
description
when to use
inputs
outputs
tool mapping
reporting contract
policy references
target profiles
```

### ToolCallPolicy

Describes safety behavior:

```text
allow
deny
mutate
requires_approval
audit
limits
forbidden paths
safe defaults
```

### TargetProfile

Describes where to generate:

```text
AGENTS.md
SKILL.md
MCP metadata
CLI/Codex/Claude/OpenCode config
OpenCode snippets
Claude snippets
policy bundle
contract tests
```

### AdoptionProfile

Describes safe behavior in existing projects:

```text
never overwrite by default
explicit apply
ownership markers
backups
remove only owned files and blocks
```

### RuntimeBinding

Optional later layer for enforcement:

```text
webhook
gateway plugin
local sidecar
MCP wrapper
native gateway policy
```

Runtime is not required for generation mode.

## Phase 0 Boundary

Phase 0 should contain documents, diagrams, spec notes, and hand-written expected artifacts.

It should not claim that a production CLI or runtime exists.
