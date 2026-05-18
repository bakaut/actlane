# Actlane Manifesto

MCP gives agents hands. Actlane defines the safe lane for every action.

AI agents now read repositories, call tools, run tests, create pull requests, write to memory, interact with CI/CD, call internal APIs, and increasingly gain the ability to change real systems.

But the rules for these actions are scattered across too many places:

```text
prompt
AGENTS.md
SKILL.md
MCP tool schema
OpenAPI Action
opencode.json
.vscode/mcp.json
gateway policy
CI scripts
README
an engineer's memory
```

The same capability lives in many formats. The same safety rule is copied by hand. The same workflow is interpreted differently by each runtime.

For read-only tasks this is inconvenient. For mutating tools it is dangerous.

## What Actlane Is

Actlane is a DSL-first system for describing portable, policy-aware capabilities for AI agents.

In one sentence:

```text
Actlane turns scattered agent prompts, tools, configs, and policies
into one typed, reusable, and auditable capability contract.
```

Actlane is not an agent, not an MCP gateway, not another IDE or runtime framework, and not a skill marketplace.

Actlane is a source of truth for safe agent actions.

## Core Idea

```text
Define safe agent actions once.
Generate agent/runtime integrations everywhere.
Optionally enforce the same contract at runtime.
```

One `actlane.yaml` can generate:

```text
AGENTS.md
SKILL.md
MCP tool metadata
MCP prompts/resources
OpenAPI Actions
OpenCode commands/config
VS Code MCP config
Codex/Cursor/Cline/Continue snippets
policy bundle
contract tests
audit schema
```

## First Focus

Actlane starts with portable private capability packs.

The first pack is `safe-gitops`.
The first workflow is safe AI-assisted draft pull request creation.

The first demonstration is:

```text
one actlane.yaml
-> AGENTS.md
-> SKILL.md
-> MCP metadata
-> OpenAPI Action
-> OpenCode / VS Code snippets
-> policy bundle
-> allow / deny / mutate examples
```

## Reproducibility

Actlane needs a lockfile:

```text
actlane.yaml = desired capability contract
actlane.lock = exact generated and applied state
```

Like `package-lock.json` makes dependency resolution reproducible, `actlane.lock` makes agent capability generation reproducible.

It records pack versions, spec versions, generator versions, adapter versions, capability digests, policy digests, generated files, owned blocks, checksums, applied targets, and audit metadata.

## Adoption Principle

Actlane should be safe to try in an existing project.

```text
inspect -> init -> generate -> plan -> apply -> remove
```

By default:

```text
generate beside the project
show the diff
apply only explicitly
remove only owned files and owned blocks
```

Install should be boring. Apply should be explicit. Remove should be trustworthy.
