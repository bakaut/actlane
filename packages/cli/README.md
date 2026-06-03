# Actlane CLI

Status: Phase 1 MVP implementation.

## Quick Start

```bash
curl -fsSL https://actlane.ru/install.sh | sh
actlane version
```

Override install options:

```bash
ACTLANE_VERSION=v0.3.0-alpha.12 ACTLANE_INSTALL_DIR="$HOME/.local/bin" sh -c "$(curl -fsSL https://actlane.ru/install.sh)"
```

Docker:

```bash
docker run --rm ghcr.io/actlane/actlane:0.3.0-alpha.12 version
```

Implemented MVP commands:

```bash
go run ./cmd/actlane version
go run ./cmd/actlane inspect
go run ./cmd/actlane import
go run ./cmd/actlane import report
go run ./cmd/actlane pack init safe-deploy --out ../../packs/safe-deploy
go run ./cmd/actlane pack init thefirm --out ../../packs/thefirm --targets codex,opencode --contracts all
go run ./cmd/actlane pack create
go run ./cmd/actlane pack inspect actlane-pack.zip
go run ./cmd/actlane pack install actlane-pack.zip --target codex
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane generate
go run ./cmd/actlane plan ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane apply ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane remove ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --frozen-lockfile
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --frozen-lockfile
go run ./cmd/actlane check --pack ../../packs/create-github-draft-pr --tool github_create_pull_request
go run ./cmd/actlane mcp serve --broker-bundle ../../packs/create-github-draft-pr/generated/codex/broker/broker-bundle.json
go run ./cmd/actlane mcp serve --policy-bundle ../../packs/create-github-draft-pr/generated/codex/policies/policy-bundle.json
go run ./cmd/actlane mcp serve --pack ../../packs/create-github-draft-pr # dev/debug source-pack path
go run ./cmd/actlane mcp author serve --pack ../../packs/create-github-draft-pr
go run ./cmd/actlane schema list
go run ./cmd/actlane schema print capability
```

`mcp serve --broker-bundle` exposes broker tools for classify, load capability, policy-gated run capability, session-local evidence lookup, and final delivery summary from generated runtime JSON.

`actlane_run_capability` keeps adapter execution disabled by default. Pass `executeAdapters: true` to execute configured stdio MCP tools, and `evidenceDir` to persist compact evidence JSON files.

From the repository root, run `./scripts/smoke-mcp-broker.sh` for a full MCP broker smoke flow with safe default, deny, durable evidence, and fake stdio adapter execution scenarios.

The MVP supports OpenCode and Codex targets.

## Brownfield OpenCode Import

For an existing OpenCode project, the shortest user workflow is:

```bash
actlane inspect
actlane import
actlane import report
actlane pack create
```

Defaults:

```text
inspect: --from . --ai-agent auto
import:  --from . --out .actlane --ai-agent auto
pack:    --from .actlane --out actlane-pack.zip
```

Another developer can generate Codex artifacts from the pack:

```bash
actlane pack inspect actlane-pack.zip
actlane generate actlane-pack.zip --target codex
```

Codex reads project-local MCP config from `.codex/config.toml` when it is run from the project root:

```bash
codex mcp list
```

Imported capability, policy, and MCP binding objects may be inferred. Review `actlane import report` before trusting the pack as a safety contract.

The source of truth for Actlane JSON Schemas is outside this Go module:

```text
../../spec/v1alpha1/schemas/
```
