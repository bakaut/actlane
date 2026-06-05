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

The generated Codex profile includes project-local instructions, skills, MCP config, policy bundle, and broker bundle artifacts. Review the `plan` output before `apply`.

## Use Case: Share Codex Config To OpenCode

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

Then Developer A sends `actlane-pack.zip` to Developer B through an approved file-transfer channel.

Then Developer B generates an OpenCode profile from the received pack:

```bash
actlane pack inspect ./actlane-pack.zip
actlane generate ./actlane-pack.zip --target opencode
```

This is the intended migration flow: `Codex config -> Actlane pack -> approved transfer -> OpenCode generation`. Environment values, hooks, credentials, auth, sessions, history, trust state, logs, caches, and SQLite state are never transferred.
