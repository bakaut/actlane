# Actlane

MCP gives agents hands. Actlane defines the safe lane for every action.

Actlane is a pre-alpha project for portable, policy-aware capability packs for AI agents. The current CLI MVP can import an existing OpenCode setup, normalize it into an Actlane pack, and generate target-specific artifacts for OpenCode and Codex.

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
ACTLANE_VERSION=v0.3.0-alpha.2 ACTLANE_INSTALL_DIR="$HOME/.local/bin" sh -c "$(curl -fsSL https://actlane.ru/install.sh)"
```

Docker:

```bash
docker run --rm ghcr.io/bakaut/actlane:0.3.0-alpha.2 version
```

## Quick Start

Start from an existing OpenCode project:

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

## Current CLI

The current MVP supports:

```bash
actlane version
actlane inspect
actlane import
actlane import report
actlane pack create
actlane pack inspect actlane-pack.zip
actlane pack install actlane-pack.zip --target codex
actlane validate <pack>
actlane generate <pack> --target opencode
actlane generate <pack> --target codex
actlane generate --target codex
actlane mcp serve --pack <pack>
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
- Import of OpenCode MCP servers and permission-derived MCP tool bindings.
- Pack zip create/inspect/install flow.
- Direct generation from `actlane-pack.zip`.
- Local MCP policy gate prototype via `actlane mcp serve`.
- Manual GitHub Actions workflows for CLI release artifacts and Docker image builds.

## What Does Not Exist Yet

- No production security guarantees.
- No hosted registry.
- No marketplace.
- No full apply/remove lifecycle.
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
