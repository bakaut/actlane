# Changelog

## [0.3.0-alpha.10] - 2026-06-03

Twelfth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.9` to `0.3.0-alpha.10` because the MCP broker now exposes `actlane_prepare_delivery` before `1.0.0`;
- keeps the `alpha` prerelease channel because external MCP adapter calls and durable evidence storage remain disabled in the MVP.

### Changed

- Added read-only `actlane_prepare_delivery` to prepare a final delivery summary from the latest or selected Actlane run.
- Delivery summaries combine evaluator output, `ResponsibilityContract` risk/human boundary decisions, compact `EvidenceContract` evidence, adapter execution records, and residual risk.
- Added session-local run storage keyed by evidence id so delivery summaries can be prepared after `actlane_run_capability`.
- Updated broker tests for classify/load/run/evidence/delivery flow and command/env non-leakage.
- Updated README, CLI README, STATUS, ADR handoff, workflow defaults, and pack metadata to `0.3.0-alpha.10`.

## [0.3.0-alpha.9] - 2026-06-03

Eleventh SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.8` to `0.3.0-alpha.9` because the MCP broker now returns adapter execution records and compact evidence summaries before `1.0.0`;
- keeps the `alpha` prerelease channel because external MCP adapter calls and durable evidence storage are still disabled in the MVP.

### Changed

- Extended `actlane_run_capability` with `adapterExecutions` derived only from `MCPBinding.requiredTools`.
- Added compact `evidence` output derived from `EvidenceContract.summaryFields`, raw-output policy, redaction flags, delivery checklist, and deterministic evidence id prefix.
- Added read-only `actlane_get_evidence` for retrieving session-local evidence by id or latest marker.
- Kept external MCP execution disabled by default; broker responses record planned adapter executions without leaking MCP server command/env.
- Updated broker tests for adapter/evidence output, deny-mode evidence, command/env non-leakage, and versioned CLI output.
- Updated README, CLI README, STATUS, workflow defaults, and pack metadata to `0.3.0-alpha.9`.

## [0.3.0-alpha.8] - 2026-06-03

Tenth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.7` to `0.3.0-alpha.8` because the broker guarded execution phase adds `actlane_run_capability` before `1.0.0`;
- keeps the `alpha` prerelease channel because adapter execution and evidence store behavior are still MVP/future phases.

### Changed

- Added policy-gated `actlane_run_capability` to `actlane mcp serve --pack <pack>`.
- `actlane_run_capability` evaluates `ToolCallPolicy` and `ResponsibilityContract` through the shared evaluator and returns original input, mutated input, policy decision, reasons, and required checks/evidence.
- Added `MCPBinding`-derived downstream planning for guarded capability runs while keeping external adapter execution non-performing in this MVP.
- Added tests proving allow/deny behavior, enforce-mode error reporting, downstream plan generation, and no downstream command/env leakage.
- Updated README, CLI README, STATUS, ADR handoff, workflow defaults, and versioned CLI outputs to `0.3.0-alpha.8`.

## [0.3.0-alpha.7] - 2026-06-03

Ninth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.6` to `0.3.0-alpha.7` because the broker capability-loading phase adds a new MCP tool and compact runtime API before `1.0.0`;
- keeps the `alpha` prerelease channel because the full broker flow is still being implemented in phases.

### Changed

- Added read-only `actlane_load_capability` to `actlane mcp serve --pack <pack>`.
- Added compact capability view derived from YAML contracts, including refs, interface, policy summary, responsibility boundary, required evidence, downstream tool summary, policy gate tools, and runtime profile summary.
- Added broker tests proving `actlane_load_capability` does not leak downstream server command/env details and does not mutate or execute tools.
- Updated README, CLI README, STATUS, workflow defaults, and versioned CLI outputs to `0.3.0-alpha.7`.

## [0.3.0-alpha.6] - 2026-06-03

Eighth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.5` to `0.3.0-alpha.6` because the broker advisory MVP adds new contract kinds and an MCP tool before `1.0.0`;
- keeps the `alpha` prerelease channel because RuntimeProfile, EvidenceContract, and broker behavior are still pre-production.

### Changed

- Added `RuntimeProfile` and `EvidenceContract` contract loading, validation, schemas, source digests, and `create-github-draft-pr` examples.
- Added `Capability.spec.runtimeRef` and `Capability.spec.evidenceRef` validation.
- Added read-only `actlane_classify` to `actlane mcp serve --pack <pack>` for advisory work type, risk, mode, candidate capability, and required evidence classification.
- Extended `pack init --contracts all` scaffold output with runtime profile and evidence contract source files.
- Updated README, CLI README, STATUS, workflow defaults, and versioned CLI outputs to `0.3.0-alpha.6`.

## [0.3.0-alpha.5] - 2026-06-02

Seventh SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.4` to `0.3.0-alpha.5` because source pack scaffolding adds a new CLI authoring workflow before `1.0.0`;
- keeps the `alpha` prerelease channel because pack contracts, target profiles, and runtime policy behavior are still pre-production.

### Changed

- Added `actlane pack init <name>` for creating a minimal valid source pack scaffold without hand-writing folders.
- Added `actlane pack init --contracts` so users can choose which source contracts to scaffold, including `all` for capability, policy, MCP binding, skill, command, agent, responsibility, and target profile files.
- Added shared scaffold templates reused by CLI `pack init` and Pack Authoring MCP plan/apply flows.
- Added pack-init coverage for Codex and OpenCode target profile scaffolds, validation, generation, and non-overwrite behavior.
- Updated README, CLI README, STATUS, workflow defaults, and versioned CLI outputs to `0.3.0-alpha.5`.

## [0.3.0-alpha.4] - 2026-05-27

Sixth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.3` to `0.3.0-alpha.4` because Codex safe adoption now carries project-local MCP config, policy bundles, and remove lifecycle behavior before `1.0.0`;
- keeps the `alpha` prerelease channel because target profile and runtime policy contracts are still pre-production.

### Changed

- Changed Codex safe adoption to apply MCP config into project-local `.codex/config.toml`.
- Added target-profile `markerStyle` support so TOML owned blocks use `# actlane:start/end` markers while Markdown keeps HTML markers.
- Added Codex safe adoption of `policies/policy-bundle.json` as an owned generated runtime input.
- Updated `actlane remove` to safely remove project-local Codex MCP config blocks and policy bundles.
- Verified `actlane mcp serve --policy-bundle` through JSON-RPC `tools/list` and `tools/call` for allow, deny, stop, and unregistered-tool cases.
- Added separate Pack Authoring MCP helper at `actlane mcp author serve --pack <pack>` with inspect, validate, plan-change, confirmed apply-change, generate-preview, and explain-errors tools.
- Added `actlane-pack-author` to the Codex MCP profile for the `create-github-draft-pr` pack.
- Fixed Codex safe adoption planning so unchanged Actlane-owned blocks are skipped instead of reported as updates forever.
- Updated README, CLI README, STATUS, workflow defaults, and versioned CLI outputs to `0.3.0-alpha.4`.

## [0.3.0-alpha.3] - 2026-05-27

Fifth SemVer-tagged CLI MVP release.

SemVer rationale:

- prerelease bump from `0.3.0-alpha.2` to `0.3.0-alpha.3` because safe adoption, removal, and release documentation changed before `1.0.0`;
- keeps the `alpha` prerelease channel because contract boundaries and generated target profiles are still pre-production.

### Changed

- Split contract ownership so each Actlane object has one primary responsibility.
- Moved target-specific generated file layout out of `Capability` and into `TargetProfile.files`.
- Added pack-level `contracts` loading for `ResponsibilityContract`.
- Added `ResponsibilityContract` schema and kind documentation.
- Added `MCPBinding` kind documentation.
- Added `responsibilityRef` support on `Capability`.
- Changed `SkillContract` to keep only portable skill body/resources; generated `SKILL.md` now derives required inputs, policy gate tools, downstream MCP tools, and reporting fields from linked YAML contracts.
- Removed duplicated OpenCode prompt markdown sources from `capabilities/`.
- Cleaned `CommandContract` and `AgentContract` so safety, output, permissions, and target projection logic stay in their owning contracts.
- Added validation checks that reject contract-boundary violations such as target paths in `Capability`, generated MCP sections in `SkillContract`, CLI lifecycle fields in `ResponsibilityContract`, and exact MCP tools outside `MCPBinding`.
- Added shared evaluator runtime for policy mutation/validation plus `ResponsibilityContract` risk, checks, evidence, human approval, stop, and MCP tool governance decisions.
- Added `actlane check` for CLI/CI evaluation from a pack or generated `policy-bundle.json`.
- Added read-only `actlane plan` for Codex safe adoption planning against existing project files, including preview metadata plus optional `--diff`, `--show-content`, and `--json` output.
- Added Codex `actlane apply <pack> --target codex` with conflict blocking, dry-run mode, create-file, append-owned-block, update-owned-block, and idempotent skip behavior.
- Added Codex `actlane remove <pack> --target codex` with dry-run mode, owned-block removal, generated-file removal, and conflict blocking for user-modified generated files.
- Changed Codex safe adoption UX so `plan` and `apply` require explicit `--target`, while generated source and project path still default from the pack and current directory.
- Added safe cleanup for generated Codex adoption output through `actlane remove`.
- Added Codex safe adoption entries for project-local `.codex/config.toml` MCP configuration and `policies/policy-bundle.json`.
- Updated release defaults, README examples, and versioned CLI outputs to `0.3.0-alpha.3`.
- Changed `actlane mcp serve` to call the same evaluator used by `actlane check` instead of owning policy evaluation logic.
- Changed generated `policy-bundle.json` to include `ResponsibilityContract` input for self-contained runtime evaluation.
- Removed GitHub-specific downstream argument shaping from the MCP server.
- Added the English `dushnila` SkillContract and generated skill outputs for OpenCode and Codex.
- Updated brownfield import to emit target layout in `TargetProfile` instead of `Capability`.
- Regenerated OpenCode and Codex outputs from the cleaned contract graph.
- Updated architecture handoff documentation in `.bakaut/.agent`.

## [0.2.0-alpha.1] - 2026-05-24

Second SemVer-tagged CLI MVP release.

SemVer rationale:

- minor bump from `0.1.0-alpha.1` to `0.2.0-alpha.1` because this release adds new CLI workflows and changes MVP behavior before `1.0.0`;
- keeps the `alpha.1` prerelease channel because contracts are still pre-production and may change.

### Changed

- Added brownfield OpenCode adoption flow: `actlane inspect`, `actlane import`, `actlane import report`, `actlane pack create`, `actlane pack inspect`, and `actlane pack install`.
- Added default-based UX so `inspect`, `import`, `pack create`, and `generate` can run without repeated path arguments.
- Added direct generation from `actlane-pack.zip`; `actlane generate --target <target>` now works without `.actlane/` when `actlane-pack.zip` exists in the current directory.
- Added OpenCode import support for `.opencode/opencode.jsonc` as well as project-root `opencode.jsonc`.
- Added MCP server and tool import from OpenCode config, including local/remote server fields and permission-derived tools such as `codegraph_*`, `memory_*`, `context7_*`, and `serena_*`.
- Added generated `MCPBinding.requiredTools` entries for imported OpenCode MCP tool permissions.
- Added `install.sh` for `curl -fsSL https://actlane.ru/install.sh | sh` installs from the latest GitHub Release on macOS and Linux.
- Added a Dockerfile and manual Docker image workflow for publishing the Actlane CLI image to GHCR.
- Expanded release binary matrix with Linux arm64 and macOS amd64 artifacts.
- Aligned the Go CLI with `packs/create-github-draft-pr` as the source of truth.
- Added support for pack-level `guidance.sources` and `guidance.compose` to generate OpenCode `AGENTS.md`.
- Added support for `mcpBindings` and generated MCP server/tool metadata from `mcp/bindings/*.yaml`.
- Updated capability parsing for `intent`, `interface`, `policyRef`, `executionRef`, `workflowHints`, and `projections`.
- Updated policy parsing for `match`, `mutate.defaults`, `mutate.ensure`, `validate`, `approval`, and `audit`.
- Updated target profile parsing for `generate` and `opencode.config`.
- Changed generated output to match the pack: `generated/opencode/AGENTS.md`, `generated/opencode/opencode.jsonc`, `generated/mcp/*`, `generated/policies/policy-bundle.json`, and `generated/actlane.lock`.
- Added source digests for guidance files and MCP bindings to `actlane.lock`.
- Added `mcp-binding.schema.json` and updated pack, capability, target profile, and policy schemas to match the current pack format.
- Updated the Go generator to emit the official OpenCode project layout under `generated/opencode`, including `.opencode/commands/*.md`, `.opencode/agents/*.md`, and `.opencode/skills/<name>/SKILL.md`.
- Changed OpenCode MCP config rendering to translate real `mcp/bindings/*.yaml` servers into `opencode.jsonc`.
- Refactored the profile generator into target-specific and shared files so OpenCode rendering is isolated from generic generation, lockfile, MCP, policy, and path logic.
- Added `actlane mcp serve --pack <pack>` with MCP `tools/list` and `tools/call` support for audit/enforce policy validation and mutation.
- Moved the `actlane-safe-gitops` local MCP server contract into `mcp/bindings/actlane-safe-gitops.yaml`.
- Updated the Actlane MCP enforce response to act as a security gate by returning mutated input plus downstream GitHub MCP `next` calls when policy allows execution.
- Updated OpenCode generation to use target-local `policies/policy-bundle.json` for safe-gitops runtime data.
- Updated pack loading so CLI commands accept both a pack directory and a direct `actlane.yaml` manifest path.
- Added Codex target generation for `create-github-draft-pr`, including `AGENTS.md`, a project-local Codex skill, and a Codex MCP config snippet.
- Moved target-specific lockfiles and policy bundles under `generated/<target>/` so OpenCode and Codex generation can coexist without overwriting each other's drift state.
- Removed generated agent/skill/command prose from Go renderers; target agent files are now emitted only from explicit `spec.profiles.<target>.files` entries in capability YAML.
- Changed `actlane-safe-gitops` generated MCP startup from `--pack ./actlane.yaml` to `--policy-bundle ./policies/policy-bundle.json`; the policy bundle now carries policy rules plus generated and downstream MCP tool bindings needed by the local gate.
- Removed obsolete runtime-pack files from generated target profiles; generated targets no longer include `actlane.yaml`, raw capability YAML, raw policy YAML, MCP binding YAML, target-profile YAML, or copied prompt sources.
- Added `kind: SkillContract` loading and rendering so generated Codex/OpenCode `SKILL.md` files are translated from YAML DSL instead of copied from profile-specific Markdown sources.
- Restricted `SkillContract` to the portable skill-directory primitive: `SKILL.md` front matter/body plus optional `scripts/`, `references`, and `assets` resources.
- Added `kind: CommandContract` loading, validation, schema support, and OpenCode command rendering from YAML instead of raw Markdown command sources.
- Added `kind: AgentContract` loading, validation, schema support, and OpenCode agent rendering from YAML instead of raw Markdown agent sources.

## [0.1.0-alpha.1] - 2026-05-21

First SemVer-tagged CLI MVP release.

SemVer policy:

- `0.x` means the CLI and schema contracts may still change.
- prerelease identifiers such as `alpha.N` mark non-production builds.
- patch releases should be backward-compatible bug fixes for the same prerelease line.
- minor releases may add or change MVP contracts before `1.0.0`.

### Added

- Added the first working Go CLI MVP under `packages/cli`.
- Added `actlane version`.
- Added `actlane validate <pack>` for local pack validation.
- Added `actlane generate <pack> --target opencode`.
- Added `actlane generate <pack> --target opencode --check` for CI drift detection.
- Added `actlane generate <pack> --target opencode --frozen-lockfile`.
- Added `actlane schema list` and `actlane schema print capability`.
- Added the first executable MVP pack: `packs/create-github-draft-pr`.
- Added OpenCode-only generation for the `create-github-draft-pr` capability.
- Added generated OpenCode artifacts:
  - `opencode/opencode.snippet.jsonc`
  - `opencode/commands/create-github-draft-pr.md`
  - `opencode/agents/github-draft-pr.md`
  - `opencode/instructions/github-draft-pr.md`
- Added generated policy bundle and `actlane.lock` for the OpenCode MVP pack.
- Added GitHub Actions manual release workflow for Linux, macOS, and Windows CLI artifacts.

### Changed

- Updated `spec/v1alpha1/schemas` to match the OpenCode MVP capability contract.
- Kept `spec/v1alpha1/schemas` as the single source of truth for schemas in git.
- Moved all Go code into `packages/cli`, including `cmd` and `internal`.
- Updated package docs to reflect that CLI MVP code now exists.

### Not Included Yet

- No production security guarantees.
- No runtime enforcement.
- No MCP gateway.
- No hosted registry.
- No marketplace.
- No project apply/remove lifecycle.
- No generated MCP, Codex, Claude, or multi-target adapters in the working CLI.
