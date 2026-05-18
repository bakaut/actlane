# Actlane Roadmap

This is the short public roadmap. Detailed planning lives in:

- [docs/01-ROADMAP.MD](docs/01-ROADMAP.MD)
- [docs/02-ROADMAP-FOLDERS.MD](docs/02-ROADMAP-FOLDERS.MD)

## Phase 0: Public RFC / Demand Validation

Goal: validate the problem before writing real implementation code.

Scope:

```text
README.md
STATUS.md
MANIFESTO.md
ROADMAP.md
CONTRIBUTING.md
SECURITY.md
LICENSE
docs/
spec/v1alpha1/
diagrams/
packs/safe-gitops/
examples/create-safe-draft-pr/
```

Phase 0 is complete when:

- README explains the pain in under 30 seconds.
- Manifesto clearly states what Actlane is and is not.
- One concrete pack exists: `safe-gitops`.
- One concrete workflow exists: `create-safe-draft-pr`.
- Example generated artifacts exist, even if hand-written.
- `STATUS.md` says pre-alpha / RFC / no production CLI yet.
- Call to action is clear: star, open issue, request target/runtime.

## Phase 1: Minimal CLI Prototype

Goal: make Actlane real enough to run locally.

Smallest useful commands:

```bash
actlane validate
actlane generate --target mcp
actlane generate --target agents
actlane generate --target policy
```

Non-goals:

- no `apply`;
- no runtime check;
- no webhook;
- no gateway plugin;
- no marketplace;
- no enterprise UI.

## Phase 2: Brownfield-Safe Adoption

Goal: let users try Actlane in an existing project without fear.

Commands to prove later:

```bash
actlane inspect
actlane init --no-touch
actlane generate
actlane diff
actlane plan apply
actlane remove --dry-run
```

## Phase 3: More Targets, Still Generation-First

Add target adapters only when they are needed by the demo or requested by users.

Priority:

```text
AGENTS.md
SKILL.md
MCP metadata
OpenCode snippets
VS Code MCP config snippets
OpenAPI Actions for Custom GPT
Codex/Cursor/Cline/Continue snippets
Orchestra pack target
```

## Phase 4: Local Policy Check

Goal: evaluate tool-call JSON against a generated policy bundle without a daemon or gateway.

Supported decisions:

```text
allow
deny
mutate
requires-approval
```

## Phase 5: Optional Runtime Enforcement

Goal: allow teams to enforce the same policy bundle at runtime if they want to.

Actlane should still work without a daemon or service.

## Phase 6: Private Pack Distribution

Goal: make capability packs portable between developers and teams.

Possible lifecycle:

```bash
actlane pack create
actlane pack inspect
actlane pack install
actlane pack remove
actlane pack verify
```

## Phase 7: Ecosystem And Enterprise Options

Only after real usage:

- hosted/private registry;
- signed packs;
- RBAC/SSO;
- audit dashboard;
- approval workflow;
- enterprise policy packs;
- MCP gateway integrations.
