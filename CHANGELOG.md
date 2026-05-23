# Changelog

## Unreleased - 2026-05-23

### Changed

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
- Updated OpenCode generation to include a self-contained runtime pack under `generated/opencode` so `actlane mcp serve --pack ./actlane.yaml` can load local capabilities, policies, MCP bindings, target profile, and prompt sources after install.
- Updated pack loading so CLI commands accept both a pack directory and a direct `actlane.yaml` manifest path.

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
