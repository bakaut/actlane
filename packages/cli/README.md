# Actlane CLI

Status: Phase 1 MVP implementation.

Implemented MVP commands:

```bash
go run ./cmd/actlane version
go run ./cmd/actlane inspect
go run ./cmd/actlane import
go run ./cmd/actlane import report
go run ./cmd/actlane pack create
go run ./cmd/actlane pack inspect actlane-pack.zip
go run ./cmd/actlane pack install actlane-pack.zip --target codex
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane generate
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --frozen-lockfile
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --frozen-lockfile
go run ./cmd/actlane mcp serve --pack ../../packs/create-github-draft-pr/generated/codex/actlane.yaml
go run ./cmd/actlane schema list
go run ./cmd/actlane schema print capability
```

The MVP supports OpenCode and Codex targets.

## Brownfield OpenCode Import

For an existing OpenCode project, Actlane can capture the native setup into `.actlane/`:

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

Another developer can install the pack for Codex:

```bash
actlane pack inspect actlane-pack.zip
actlane pack install actlane-pack.zip --target codex
actlane generate
```

`pack install --target codex` writes `.actlane/.local.yaml`, so `generate` can resolve the default target without `--target`.

Imported capability, policy, and MCP binding objects may be inferred. Review `actlane import report` before trusting the pack as a safety contract.

The source of truth for Actlane JSON Schemas is outside this Go module:

```text
../../spec/v1alpha1/schemas/
```
