# ADR-0001: Keep the Codex Development Automation Surface Small

## Status

Accepted

## Date

2026-06-09

## Context

Actlane has recurring development workflows around architecture planning,
contract-driven implementation, generated artifacts, verification, release
versioning, changelog maintenance, and publication.

These workflows can be exposed to Codex as skills and commands. Exposing every
individual step would make the automation difficult to discover and would
create overlapping responsibilities. Hiding too much would make important
project rules easy to lose.

The automation surface therefore needs to stay small while preserving the full
development and release workflow.

## Decision

Expose three primary Codex skills and five primary commands.

Codex does not support repository-scoped custom slash prompts. Implement the
five commands as repository-scoped explicit-invocation skills under
`.agents/skills`. Prefix every skill name with `actlane-` to avoid collisions
with user or global skills. Keep `/architect`, `/implement`, `/verify`,
`/finish`, and `/release` as logical command names used by this ADR and its
diagrams; in Codex, invoke them as `$actlane-command-architect`,
`$actlane-command-implement`, `$actlane-command-verify`,
`$actlane-command-finish`, and `$actlane-command-release`.

### Primary Skills

#### `actlane-architect`

Responsibilities:

- load relevant repository context;
- separate facts, assumptions, decisions, risks, and open questions;
- define boundaries, contracts, flow, and engineer handoff;
- avoid production implementation.

Reason:

- architecture and scope control must remain distinct from implementation;
- it combines context preparation, contract analysis, and engineer handoff
  without exposing them as separate skills.

#### `actlane-engineer`

Responsibilities:

- implement an approved architect handoff;
- preserve YAML contracts as the authoring source of truth;
- update schemas, validation, generators, fixtures, and related documentation;
- follow minimal-pack, `packs/full`, broker, and safe-adoption patterns;
- avoid expanding scope without confirmation.

Reason:

- implementation work crosses several repository layers;
- one engineering skill preserves end-to-end ownership better than multiple
  overlapping implementation skills.

#### `actlane-release-manager`

Responsibilities:

- determine the SemVer bump and rationale;
- update versions, changelog, documentation, packs, and workflow defaults;
- verify release readiness;
- publish only after explicit confirmation.

Reason:

- releases require coordinated changes across many files;
- keeping release ownership separate reduces version and changelog drift.

### Primary Commands

#### `/architect <task>`

- invoke `actlane-architect`;
- prepare context, architecture analysis, and engineer handoff.

#### `/implement`

- invoke `actlane-engineer`;
- implement the approved handoff;
- regenerate outputs and update related fixtures and documentation.

#### `/verify`

- run Go tests;
- validate the minimal reference pack and `packs/full`;
- run Codex and OpenCode generation drift checks;
- run broker smoke tests when the change affects runtime or MCP behavior.

#### `/finish`

- review scope and diff;
- require successful verification;
- check README, STATUS, examples, and generated artifacts;
- report unresolved questions and residual risks.

#### `/release <version|auto>`

- invoke `actlane-release-manager`;
- choose or validate the version;
- update changelog and all version references;
- run verification and stale-version search;
- prepare publication and publish only after confirmation.

### Standard Flow

```text
/architect -> /implement -> /verify -> /finish -> /release auto
```

Commands may be used independently when their preconditions are already
satisfied.

## Parked Skills

- `actlane-contract-author`: covered by `actlane-engineer`; reconsider when the
  DSL and schema lifecycle require dedicated ownership.
- `actlane-generator-change`: covered by `actlane-engineer`; reconsider when
  target adapters have substantially different development workflows.
- `actlane-pack-maintainer`: covered by implementation and verification;
  reconsider when the maintained pack set grows significantly.
- `actlane-safe-adoption`: covered by engineering patterns; reconsider when
  apply/remove lifecycle support expands across more targets.
- `actlane-mcp-broker`: covered by engineering patterns and conditional smoke
  testing; reconsider when broker/runtime work becomes an independent stream.
- `actlane-regression-verifier`: replaced by `/verify`.
- `actlane-doc-sync`: replaced by the documentation gate in `/finish`.
- `actlane-engineer-handoff`: included in `actlane-architect`.

## Parked Commands

- `/context-prepare`: internal stage of `/architect`.
- `/change-start`: internal stage of `/architect`.
- `/handoff-engineer`: output of `/architect`.
- `/pack-validate`: part of `/verify`.
- `/regenerate`: part of `/implement` and `/verify`.
- `/drift-check`: part of `/verify`.
- `/verify-change`: replaced by `/verify`.
- `/docs-sync`: part of `/finish`.
- `/finish-change`: replaced by `/finish`.
- `/release-prepare`: part of `/release`.
- `/release-check`: part of `/release`.
- `/release-publish`: confirmation-gated stage of `/release`.

## Consequences

Positive:

- users choose from a small and predictable automation surface;
- responsibilities remain distinct between architecture, implementation, and
  release management;
- verification and documentation requirements remain mandatory;
- internal workflow steps can evolve without changing public commands.

Negative:

- primary skills have broader internal responsibilities;
- individual internal steps are less directly invokable;
- command implementations must preserve clear preconditions and reporting.

## Reconsideration Triggers

Revisit this decision when:

- a parked workflow repeatedly needs independent invocation;
- one primary skill becomes too large to maintain or reason about;
- new target adapters require materially different implementation flows;
- release publication becomes independent from release preparation;
- users cannot diagnose failures because internal stages are insufficiently
  visible.
