# Actlane Concept

Actlane is a DSL-first system for portable, policy-aware AI-agent capabilities.

The core idea:

```text
Define safe agent actions once.
Generate agent/runtime integrations everywhere.
Optionally enforce the same contract at runtime.
```

## Source Of Truth

Actlane starts from a capability contract:

```text
actlane.yaml
capabilities/*.yaml
policies/*.yaml
target-profiles/*.yaml
```

The contract describes a high-level action such as:

```text
create-safe-draft-pr
```

Instead of exposing raw tools like:

```text
github.create_branch
github.commit_files
github.create_pull_request
sandbox.run_tests
```

Actlane describes the safe workflow around them.

## Generated Artifacts

The same contract can generate:

```text
AGENTS.md
SKILL.md
MCP metadata
Claude/Codex/OpenCode adapter
OpenCode commands/config
Claude instruction snippets
policy bundle
contract tests
audit schema
```

## Reproducibility

Actlane uses a lockfile model:

```text
actlane.yaml = desired capability contract
actlane.lock = exact generated and applied state
```

The lockfile should record generated files, owned blocks, checksums, pack versions, policy digests, target adapter versions, and audit metadata.

## First Pack

The first practical pack is:

```text
safe-gitops
```

The first workflow is:

```text
create-safe-draft-pr
```

This keeps the MVP narrow and useful.
