# Brownfield Adoption

Actlane must be safe to try in an existing project.

The long-term adoption lifecycle is:

```text
inspect -> init -> generate -> diff -> plan -> apply -> remove
```

The current MVP starts with a narrower portability flow:

```text
existing OpenCode project
-> actlane inspect
-> actlane import
-> .actlane as source of truth
-> actlane pack create
-> actlane pack install --target codex
-> actlane generate
```

## Default Behavior

By default, Actlane should:

```text
inspect existing files
create .actlane/ only
generate into generated/<target> inside the pack
show diffs before apply
avoid overwriting files
write only owned blocks
record ownership metadata
remove only owned files and blocks
```

For the import MVP, `inspect` is always read-only. `import` writes a normalized `.actlane/` pack and does not mutate `.opencode/`, `.codex/`, `AGENTS.md`, or project runtime files.

## OpenCode Import MVP

The first brownfield case is a developer who already has a working OpenCode setup but no Actlane pack.

Example source project:

```text
project/
  AGENTS.md
  opencode.jsonc
  .opencode/
    commands/*.md
    agents/*.md
    skills/*/SKILL.md
```

The intended flow is:

```bash
actlane inspect
actlane import
actlane import report
actlane pack create
```

`actlane inspect` detects the agent runtime, commands, agents, skills, MCP servers, and permissions without writing files. If detection is ambiguous, the user can select a runtime explicitly:

```bash
actlane inspect --ai-agent opencode
actlane import --ai-agent opencode
```

`actlane import` defaults to:

```text
--from .
--out .actlane
--ai-agent auto
```

After import, `.actlane/` becomes the source of truth. Native OpenCode files are treated as imported evidence, not as the continuing canonical model.

The generated `.actlane/` pack should contain:

```text
.actlane/
  actlane.yaml
  import.report.md
  actlane.lock
  capabilities/
  commands/
  agents/
  skills/
  policies/
  mcp/bindings/
  target-profiles/
```

## Inferred Objects

Import cannot reliably prove every capability, policy, or MCP execution rule from Markdown and runtime config alone.

Objects inferred by Actlane must be marked explicitly:

```yaml
metadata:
  annotations:
    actlane.ru/imported-from: opencode
    actlane.ru/import-confidence: medium
    actlane.ru/import-source: .opencode/commands/create-github-draft-pr.md
    actlane.ru/inferred: "true"
```

This is required for especially sensitive objects:

- `Capability`
- `ToolCallPolicy`
- `MCPBinding`

The import report must show what was imported directly, what was inferred, and what requires review.

## Pack Handoff

After import, the developer can create a portable pack:

```bash
actlane pack create
```

This defaults to:

```text
--from .actlane
--out actlane-pack.zip
```

Another developer can inspect and install the pack for a different target:

```bash
actlane pack inspect actlane-pack.zip
actlane pack install actlane-pack.zip --target codex
actlane generate
```

`pack install --target codex` writes the selected target into `.actlane/.local.yaml`, so `actlane generate` can run without repeating `--target`.

Actlane must not convert OpenCode files directly into Codex files. The conversion path is:

```text
OpenCode files -> DiscoveryModel -> Actlane Pack -> TargetProfile renderer
```

## Ownership Markers

If Actlane writes into an existing file, the content must be inside owned markers:

```markdown
<!-- actlane:start create-safe-draft-pr -->
Generated content
<!-- actlane:end create-safe-draft-pr -->
```

## State Files

Future adoption state may live in:

```text
.actlane/state/
.actlane/backups/
.actlane/reports/
actlane.lock
```

## Remove Must Be Trustworthy

Removal should never delete arbitrary project content.

It may remove:

- files listed as Actlane-owned in the lockfile;
- blocks marked with Actlane ownership markers;
- generated files under `.actlane/generated`.

It should refuse or warn when a generated file has been manually changed.
