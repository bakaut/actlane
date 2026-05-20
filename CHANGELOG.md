# Changelog

## Unreleased

No unreleased changes yet.

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
- Added the first executable MVP pack: `packs/github-draft-pr-opencode`.
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
