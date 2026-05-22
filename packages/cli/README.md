# Actlane CLI

Status: Phase 1 MVP implementation.

Implemented MVP commands:

```bash
go run ./cmd/actlane version
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --frozen-lockfile
go run ./cmd/actlane schema list
go run ./cmd/actlane schema print capability
```

The MVP supports only the OpenCode target.

The source of truth for Actlane JSON Schemas is outside this Go module:

```text
../../spec/v1alpha1/schemas/
```
