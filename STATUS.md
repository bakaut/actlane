# Actlane Status

Status: pre-alpha / working CLI MVP / no production security guarantees.

Actlane started as a design and specification repository. It now also contains a narrow Go CLI MVP for one OpenCode capability.

## What Exists

- Public narrative and manifesto.
- Phase-by-phase roadmap.
- Phase 0 diagram sources and SVGs.
- Early `spec/v1alpha1` notes.
- Documentation for the first intended pack: `safe-gitops`.
- Hand-written `safe-gitops` pack artifacts and examples.
- Working Go CLI MVP in `packages/cli`.
- `validate`, `generate --target opencode`, `--check`, `--frozen-lockfile`, and schema inspection commands.
- First executable MVP pack: `packs/create-github-draft-pr`.
- Generated OpenCode command, agent instructions, config snippet, policy bundle, and `actlane.lock`.
- Manual GitHub Actions release workflow for Linux, macOS, and Windows CLI artifacts.

## What Does Not Exist Yet

- No production-ready CLI contract.
- No runtime service.
- No MCP gateway.
- No hosted registry.
- No marketplace.
- No production security guarantees.
- No apply/remove lifecycle for existing projects.
- No generated MCP, Codex, Claude, or multi-target adapters in the working CLI.

## Intended Next Step

Stabilize the Phase 1 MVP:

```text
packages/cli
packs/create-github-draft-pr
spec/v1alpha1/schemas
.github/workflows/manual-build-cli.yml
```

Current validation path:

```bash
cd packages/cli
go test ./...
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --frozen-lockfile
```
