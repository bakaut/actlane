# ADR-0002: Provide a Safe Codex to OpenCode Migration Facade

## Status

Accepted

## Date

2026-06-11

## Context

Actlane can already inspect and import Codex configuration, generate an
OpenCode target, build an ownership-aware plan, apply it, and safely remove
unchanged owned artifacts. Requiring users to coordinate every internal step is
unnecessary for the common local-project migration case.

Safe removal of whole owned files requires the exact imported pack and generated
target snapshot used during apply. A state-only receipt is not sufficient under
the current ownership contract.

## Decision

Add `actlane migrate opencode` as a facade over the existing
`inspect -> import -> generate -> plan -> apply` pipeline.

The MVP:

- supports only project-local Codex to OpenCode migration;
- imports project-local objects without selecting global inventory;
- performs import and generation in an operating-system temporary directory;
- shows the ownership-aware plan before a single confirmation;
- blocks conflicts before confirmation and apply;
- supports `--dry-run`, `--diff`, `--yes`, `--json`, `--project`,
  `--from-agent`, and `--to-agent`;
- preserves the successful imported pack and generated target under
  `.actlane/migrations/codex-to-opencode/`;
- leaves lower-level migration commands available.

Use explicit `--from-agent` and `--to-agent` names because existing commands use
`--from` for filesystem paths.

The existing remove engine consumes the preserved snapshot:

```bash
actlane remove .actlane/migrations/codex-to-opencode --target opencode
```

## Consequences

- The common migration requires one command and one confirmation.
- Intermediate pack and generated directories are hidden from the project root.
- Existing conflict protection and ownership rules remain authoritative.
- The hidden snapshot is durable project state and must be retained for safe
  removal.
- Repeated migration update, migration status, reverse migration, state-only
  receipts, and broader cross-agent migration remain future work.
- Global objects, credentials, environment values, hooks, sessions, history,
  trust state, logs, caches, and other personal runtime state are not migrated.
