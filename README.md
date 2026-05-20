# Actlane

MCP gives agents hands. Actlane defines the safe lane for every action.

Actlane is a pre-alpha RFC for portable, policy-aware capability packs for AI agents. It is not a production CLI, hosted service, MCP gateway, or agent framework yet.

## The Problem

AI-agent safety rules are scattered across prompts, `AGENTS.md`, `SKILL.md`, MCP tool schemas, CLI/Codex/Claude/OpenCode configs, agent configs, gateway policies, CI scripts, and human memory.

That makes one capability hard to move, review, reproduce, and safely remove.

Actlane proposes one source of truth:

```text
actlane.yaml = desired capability contract
actlane.lock = exact generated/applied state
```

From that contract, Actlane can generate:

```text
AGENTS.md
SKILL.md
MCP metadata
CLI/Codex/Claude/OpenCode configs
IDE / agent snippets
policy bundles
contract tests
audit metadata
```

## First Practical Focus

The first pack is:

```text
safe-gitops
```

The first workflow is:

```text
create-safe-draft-pr
```

The minimal proof:

```text
one actlane.yaml
one actlane.lock
generated AGENTS.md
generated SKILL.md
generated MCP metadata
generated policy bundle
allow / deny / mutate examples
```

## What Actlane Is Not

- Not an agent.
- Not an MCP gateway.
- Not a hosted SaaS.
- Not a marketplace-first project.
- Not a replacement for `AGENTS.md`, `SKILL.md`, MCP, or agent frameworks.

## Current Status

This repository is currently documentation-first:

- manifesto;
- roadmap;
- proposed folder structure;
- PlantUML diagrams;
- early v1alpha1 spec notes;
- no production CLI yet.

See [STATUS.md](STATUS.md).

## Repository Map

```text
docs/      concept, architecture, adoption, runtime, and pack docs
spec/      proposed v1alpha1 specification
packs/     Phase 0 safe-gitops pack example
examples/  minimal and create-safe-draft-pr expected outputs
diagrams/  PlantUML sources and rendered SVGs
assets/    placeholder brand and image assets
packages/  planned package boundaries, no implementation yet
raw/       source research / exported design conversation
```

## Start Here

- [MANIFESTO.md](MANIFESTO.md)
- [ROADMAP.md](ROADMAP.md)
- [docs/00-problem.md](docs/00-problem.md)
- [docs/01-concept.md](docs/01-concept.md)
- [docs/02-architecture.md](docs/02-architecture.md)

## We need your voice

Star the repo if your team also struggles with scattered agent prompts, MCP configs, skills, and policies.

Open an issue if you want Actlane to generate packs for a specific target such as Codex, Claude, OpenCode, or CLI agents.

Use the `Try Actlane on my setup` issue template. The first real adapters should be driven by actual user setups, not guesses.
