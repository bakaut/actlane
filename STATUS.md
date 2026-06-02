# Actlane Status

Status: pre-alpha / working CLI MVP / no production security guarantees.

Actlane started as a design and specification repository. It now also contains a narrow Go CLI MVP for one OpenCode/Codex capability.

## What Exists

- Public narrative and manifesto.
- Phase-by-phase roadmap.
- Phase 0 diagram sources and SVGs.
- Early `spec/v1alpha1` notes.
- Documentation for the first intended pack: `safe-gitops`.
- Hand-written `safe-gitops` pack artifacts and examples.
- Working Go CLI MVP in `packages/cli`.
- `inspect`, `import`, `import report`, `pack init`, `pack create`, `pack inspect`, `pack install`, `validate`, `generate`, `plan`, `apply`, `remove`, `check`, `mcp serve`, `mcp author serve`, `--check`, `--frozen-lockfile`, and schema inspection commands.
- First executable MVP pack: `packs/create-github-draft-pr`.
- Minimal source pack scaffold creation via `actlane pack init <name>`.
- Generated OpenCode and Codex artifacts, target-local policy bundles, and `actlane.lock`.
- Codex safe adoption into `.codex/skills`, `.codex/config.toml`, `AGENTS.md`, and `policies/policy-bundle.json`.
- Local MCP policy evaluator via `actlane mcp serve --policy-bundle <policy-bundle.json>`.
- Local MCP pack authoring helper via `actlane mcp author serve --pack <pack>` for inspect, validate, plan, confirmed apply, preview, and error explanation.
- Brownfield OpenCode import into `.actlane/` with inferred capability, policy, MCP binding, command, agent, skill, target profile, report, and lockfile artifacts.
- Manual GitHub Actions release workflow for Linux, macOS, and Windows CLI artifacts.

## What Does Not Exist Yet

- No production-ready CLI contract.
- No production runtime service.
- No production MCP gateway.
- No hosted registry.
- No marketplace.
- No production security guarantees.
- No apply/remove lifecycle for every target yet.
- No Claude or broad multi-target adapter matrix in the working CLI.
- No registry-backed pack install/apply lifecycle yet.

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
go run ./cmd/actlane inspect --from ../../packs/create-github-draft-pr/generated/opencode
go run ./cmd/actlane pack init demo-pack --out /tmp/actlane-demo-pack --targets codex,opencode
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target codex --check
go run ./cmd/actlane plan ../../packs/create-github-draft-pr --target codex
go run ./cmd/actlane mcp serve --policy-bundle ../../packs/create-github-draft-pr/generated/codex/policies/policy-bundle.json
go run ./cmd/actlane mcp author serve --pack ../../packs/create-github-draft-pr
```
