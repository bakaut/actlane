# Actlane CLI

Status: Phase 1 MVP implementation.

Implemented MVP commands:

```bash
go run ./cmd/actlane version
go run ./cmd/actlane validate ../../packs/github-draft-pr-opencode
go run ./cmd/actlane generate ../../packs/github-draft-pr-opencode --target opencode
go run ./cmd/actlane generate ../../packs/github-draft-pr-opencode --target opencode --check
go run ./cmd/actlane generate ../../packs/github-draft-pr-opencode --target opencode --frozen-lockfile
go run ./cmd/actlane schema list
go run ./cmd/actlane schema print capability
```

The MVP supports only the OpenCode target.

The source of truth for Actlane JSON Schemas is outside this Go module:

```text
../../spec/v1alpha1/schemas/
```
