# Actlane CLI

Status: Phase 1 MVP implementation.

Implemented MVP commands:

```bash
go run ./cmd/actlane version
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --frozen-lockfile
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --frozen-lockfile
go run ./cmd/actlane mcp serve --pack ../../packs/create-github-draft-pr/generated/codex/actlane.yaml
go run ./cmd/actlane schema list
go run ./cmd/actlane schema print capability
```

The MVP supports OpenCode and Codex targets.

The source of truth for Actlane JSON Schemas is outside this Go module:

```text
../../spec/v1alpha1/schemas/
```
