# Actlane

MCP gives agents hands. Actlane defines the safe lane for every action.

Actlane is a pre-alpha project for portable, policy-aware capability packs for AI agents. The current CLI MVP can import an existing OpenCode setup, normalize it into an Actlane pack, generate target-specific artifacts for OpenCode and Codex, and safely adopt generated Codex files into a repository with `plan`, `apply`, and `remove`.

Actlane is not a production security control, hosted service, MCP gateway, marketplace, or agent framework.

## Why

AI-agent safety rules are usually scattered across prompts, `AGENTS.md`, `SKILL.md`, MCP tool configs, OpenCode/Codex/Claude settings, CI scripts, gateway policies, and human memory.

That makes one capability hard to move, review, reproduce, and safely remove.

Actlane proposes one portable source of truth:

```text
actlane.yaml = capability pack manifest
actlane.lock = generated/applied state
```

From that source, Actlane can generate:

```text
AGENTS.md
SKILL.md
OpenCode commands, agents, skills, and config
Codex skills and MCP config snippets
MCP server/tool metadata
policy bundles
audit metadata
```

## Install

```bash
curl -fsSL https://actlane.ru/install.sh | sh
actlane version
```

Install options:

```bash
ACTLANE_VERSION=v0.3.0-alpha.8 ACTLANE_INSTALL_DIR="$HOME/.local/bin" sh -c "$(curl -fsSL https://actlane.ru/install.sh)"
```

Docker:

```bash
docker run --rm ghcr.io/actlane/actlane:0.3.0-alpha.8 version
```

## Quick Start

Try the bundled MVP pack in this repository:

```bash
actlane generate ./packs/create-github-draft-pr --target codex
actlane plan ./packs/create-github-draft-pr --target codex
actlane apply ./packs/create-github-draft-pr --target codex
actlane remove ./packs/create-github-draft-pr --target codex
```

Create a new minimal source pack without hand-writing the folder structure:

```bash
actlane pack init safe-deploy
actlane validate ./packs/safe-deploy
actlane generate ./packs/safe-deploy --target codex
```

Create a fuller scaffold with command, agent, and responsibility contracts:

```bash
actlane pack init thefirm --targets codex,opencode --contracts all
actlane plan ./packs/thefirm --target codex --project .
```

Capture an existing OpenCode project into a portable pack zip:

```bash
actlane inspect
actlane import
actlane pack create
```

This creates:

```text
actlane-pack.zip
```

Generate Codex artifacts from the zip without keeping `.actlane/` in the project:

```bash
actlane generate --target codex
```

Actlane will use `./actlane-pack.zip` as the source when `.actlane/actlane.yaml` is not present.

For Codex adoption, `actlane apply <pack> --target codex` writes project-local MCP config to `.codex/config.toml`. Run Codex from the project root and it will include the local MCP servers:

```bash
codex mcp list
```

## Current CLI

The current MVP supports:

```bash
actlane version
actlane inspect
actlane import
actlane import report
actlane pack init <name> [--contracts capability,policy,mcp,skill,target-profile]
actlane pack create
actlane pack inspect actlane-pack.zip
actlane pack install actlane-pack.zip --target codex
actlane validate <pack>
actlane generate <pack> --target opencode
actlane generate <pack> --target codex
actlane generate --target codex
actlane generate <pack> --target codex --check
actlane check --pack <pack> --tool github_create_pull_request
actlane plan ./packs/create-github-draft-pr --target codex
actlane apply ./packs/create-github-draft-pr --target codex
actlane remove ./packs/create-github-draft-pr --target codex
actlane mcp serve --policy-bundle <policy-bundle.json>
actlane mcp serve --pack <pack>
actlane mcp serve --pack <pack> # exposes actlane_classify, actlane_load_capability, and actlane_run_capability
actlane mcp author serve --pack <pack>
actlane schema list
actlane schema print capability
```

`actlane import report`, `validate`, and `pack inspect` are review helpers. The shortest happy path is:

```bash
actlane inspect
actlane import
actlane pack create
actlane generate --target codex
```

## What Exists

- Go CLI in `packages/cli`.
- v1alpha1 YAML contracts and JSON Schemas in `spec/v1alpha1`.
- First working pack: `packs/create-github-draft-pr`.
- OpenCode target generation.
- Codex target generation.
- Brownfield OpenCode import into Actlane contracts.
- Source pack scaffold creation via `actlane pack init <name>`.
- Runtime and evidence contracts for advisory broker classification.
- `actlane_classify`, `actlane_load_capability`, and policy-gated `actlane_run_capability` MCP broker tools for packs with runtime/evidence contracts.
- Import of OpenCode MCP servers and permission-derived MCP tool bindings.
- Pack zip create/inspect/install flow.
- Direct generation from `actlane-pack.zip`.
- Local MCP policy gate prototype via `actlane mcp serve`.
- Local pack authoring MCP helper via `actlane mcp author serve`.
- Manual GitHub Actions workflows for CLI release artifacts and Docker image builds.

## What Does Not Exist Yet

- No production security guarantees.
- No hosted registry.
- No marketplace.
- No full apply/remove lifecycle for every target yet.
- No Claude target implementation.
- No stable `1.0` contract compatibility promise.

## Run From Source

```bash
cd packages/cli
go test ./...
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --check
```

## Repository Map

```text
docs/      concept, architecture, adoption, runtime, and pack docs
spec/      v1alpha1 schema source of truth
packs/     working MVP pack and pack examples
examples/  earlier examples and expected outputs
diagrams/  PlantUML sources and rendered SVGs
assets/    placeholder brand and image assets
packages/  Go CLI and future package boundaries
```

## Start Here

- [STATUS.md](STATUS.md)
- [CHANGELOG.md](CHANGELOG.md)
- [MANIFESTO.md](MANIFESTO.md)
- [ROADMAP.md](ROADMAP.md)
- [docs/00-problem.md](docs/00-problem.md)
- [docs/01-concept.md](docs/01-concept.md)
- [docs/02-architecture.md](docs/02-architecture.md)
- [docs/04-brownfield-adoption.md](docs/04-brownfield-adoption.md)
- [docs/09-responsibility-contract.md](docs/09-responsibility-contract.md)

## Feedback

Open an issue if you want Actlane to import or generate packs for a specific agent runtime such as Codex, Claude, OpenCode, or CLI agents.

Use the `Try Actlane on my setup` issue template. The first real adapters should be driven by actual user setups, not guesses.

![Actlane overview](diagrams/svg/actlane-overview.svg)
