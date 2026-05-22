# Changelog

## Unreleased - 2026-05-21

### Changed

- Changed the Go generator to translate capability YAML into profile files/config instead of hardcoding GitHub draft PR templates.
- Replaced target-specific `spec.opencode.profile` with generic `spec.profiles.<target>`.
- Moved large generated profile text out of capability YAML into adjacent source files referenced by `profiles.<target>.files[].source`.
- Flattened `capabilities/` so reusable profile source files live beside the capability YAML instead of under nested target folders.
- Loaded `target-profiles/*.yaml` as target contracts that define output root, config path, and profile transforms.
- Added source file digests to `actlane.lock` so `--frozen-lockfile` detects profile text drift.
- Added generated OpenCode profile files under `generated/opencode`: `AGENT.MD`, `AGENTS.md`, `SKILLS.MD`, `.opencode/agents/`, `.opencode/commands/`, and `.opencode/skills/<name>/SKILL.md`.
- Removed legacy generated OpenCode files: `opencode.snippet.jsonc`, `opencode/agents/`, `opencode/commands/`, and `opencode/instructions/`.
- Extended `spec.mcp.servers` to support OpenCode local and remote MCP server config fields.
- Added `spec.profiles.<target>` to the capability schema and generalized target/profile schemas beyond OpenCode-only constants.
- Added official GitHub MCP server bindings for `create_branch`, `push_files`, and `create_pull_request`.
- Changed `opencode.jsonc` generation to derive MCP config from `spec.mcp.servers`.

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
