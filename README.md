# Actlane

**MCP gives agents hands. Actlane defines the safe lane for every action.**

Actlane is a pre-alpha open-source CLI for teams experimenting with AI coding agents.

It helps turn messy agent setup — `AGENTS.md`, `SKILL.md`, MCP configs, OpenCode/Codex rules, prompts, and human conventions — into small, reviewable capability packs.

The first practical goal is simple:

> Let an AI agent create a safe GitHub draft PR without giving it broad raw tool access.

Actlane is currently focused on:

- importing existing OpenCode and Codex setup;
- generating Codex/OpenCode artifacts;
- keeping generated files reviewable;
- supporting `plan`, `apply`, and `remove`;
- experimenting with policy-gated MCP actions.

Actlane is **not** a production security product yet.  
It is not a hosted service, marketplace, MCP gateway, or agent framework.

It is a small attempt to make AI-agent adoption less magical and more reviewable.

## Quick Start Install

```bash
curl -fsSL https://actlane.ru/install.sh | sh
actlane version
```

## Use Case: Safe Draft PR Pack

Try the bundled pack that teaches an agent how to create a safe GitHub draft PR:

```bash
actlane validate ./packs/create-github-draft-pr
actlane generate ./packs/create-github-draft-pr --target codex
actlane plan ./packs/create-github-draft-pr --target codex --project .
actlane apply ./packs/create-github-draft-pr --target codex --project .
```

Use `--target opencode` to safely plan and apply the same pack for OpenCode.
An existing user-owned `opencode.jsonc` is reported as a conflict instead of
being overwritten or automatically merged. Review the `plan` output before
`apply`.

## Use Case: Share Codex Config To OpenCode

For a local project, migrate project-local Codex configuration to OpenCode with
one reviewed apply:

```bash
actlane migrate opencode
```

Use `--dry-run` to preview without mutation, `--diff` for detailed changes, or
`--yes` for non-interactive apply. The successful migration snapshot is kept at
`.actlane/migrations/codex-to-opencode/` so unchanged Actlane-owned OpenCode
files can be safely removed later:

```bash
actlane remove .actlane/migrations/codex-to-opencode --target opencode
```

The migration facade transfers project-local objects only. Global objects,
credentials, hooks, sessions, history, and other personal runtime state remain
excluded.

Developer A captures the current Codex agent setup into an Actlane pack:

```bash
actlane inspect --ai-agent codex
actlane import --ai-agent codex
actlane pack create
```

Global Codex skills and MCP servers are inventory-only by default. Import selected portable candidates explicitly:

```bash
actlane import --ai-agent codex \
  --include-global-skill code-review \
  --include-global-mcp github
```

Codex project skills are discovered and safely adopted under
`.agents/skills`. Actlane still reads legacy `.codex/skills` installations for
import compatibility and reports a warning; newly generated Codex profiles use
`.agents/skills`.

Then Developer A sends `actlane-pack.zip` to Developer B through an approved file-transfer channel.

Then Developer B inspects, materializes, reviews, and applies an OpenCode
profile from the received pack:

```bash
actlane pack inspect ./actlane-pack.zip
actlane generate ./actlane-pack.zip --target opencode
actlane plan ./actlane-pack.zip --target opencode
actlane apply ./actlane-pack.zip --target opencode
# for safe cleanup
actlane remove ./actlane-pack.zip --target opencode
```

`generate` writes only `./generated/opencode/`. `plan` reads that staging
directory without mutation, and `apply` is the only step that writes OpenCode
project files. `remove` reads the same staging directory and removes only
unchanged Actlane-owned files and blocks.

This is the intended migration flow: `Codex config -> Actlane pack -> approved transfer -> OpenCode generation -> reviewed apply`. Environment values, hooks, credentials, auth, sessions, history, trust state, logs, caches, and SQLite state are never transferred.
