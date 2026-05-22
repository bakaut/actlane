# Contributing To Actlane

Actlane is currently a pre-alpha repository with a narrow Go CLI MVP. The most useful contributions are clear examples, target requests, spec feedback, CLI test cases, and corrections to confusing documentation.

## Good First Contributions

- Explain a real agent workflow you want to make portable.
- Request a target runtime: Codex, Claude, OpenCode, or CLI agents.
- Open a `Try Actlane on my setup` issue with your current files and desired generated artifact.
- Improve the `safe-gitops` example.
- Improve the `create-github-draft-pr` MVP pack.
- Add policy examples for allow, deny, mutate, and requires-approval.
- Point out where the docs overpromise implementation that does not exist yet.

## Contribution Rules

- Keep the project generation-first.
- Keep runtime enforcement optional.
- Do not position Actlane as an MCP gateway.
- Do not add marketplace, SaaS, or enterprise scope to early phases.
- Prefer one concrete workflow over broad abstractions.
- Do not add commands to docs unless they exist or are clearly marked as planned.

## Phase 0 Documentation Style

Phase 0 documents should be:

- explicit about status;
- concrete about the first workflow;
- conservative about future capabilities;
- easy to read without knowing the whole repository;
- careful about safety claims.

## Development Status

There is a working CLI MVP in `packages/cli`, but it is not production-ready.

Before opening implementation changes, run:

```bash
cd packages/cli
go test ./...
go run ./cmd/actlane validate ../../packs/create-github-draft-pr
go run ./cmd/actlane generate ../../packs/create-github-draft-pr --target opencode --check
```

Keep new implementation work scoped to the current MVP unless the roadmap explicitly calls for broader target or runtime work.
