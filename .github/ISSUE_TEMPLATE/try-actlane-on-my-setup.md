# GitHub Issue Template: Try Actlane on My Setup

````markdown
---
name: Try Actlane on my setup
about: Share your current AI-agent setup so Actlane can prioritize real targets and import/export flows.
title: "Use case request: "
labels: ["use-case-request", "target-request", "phase-0-feedback"]
---

## 1. What do you use today?

Which agent/runtime do you use?

- [ ] Codex
- [ ] OpenCode
- [ ] Claude Code
- [ ] VS Code MCP / Copilot
- [ ] Cursor
- [ ] Cline
- [ ] Continue
- [ ] Other:

## 2. What is your current starting point?

- [ ] I already have a working agent setup
- [ ] I have only `AGENTS.md`
- [ ] I have one or more `SKILL.md` files
- [ ] I have MCP servers configured
- [ ] I have commands/prompts
- [ ] I have agents/subagents
- [ ] I have gateway/security policies
- [ ] I am starting from scratch

## 3. What do you want Actlane to do first?

- [ ] Generate a new capability pack from a template
- [ ] Import my existing setup into an Actlane pack
- [ ] Convert one agent setup to another, for example OpenCode → Codex
- [ ] Generate agent-specific files from one pack
- [ ] Validate policies and safety rules
- [ ] Scan existing agent configs for risky capabilities
- [ ] Other:

## 4. Workflow I want to make portable or safer

Describe the agent action.

Example:

```text
Create a safe GitHub draft PR from reviewed changes.
````

Is this workflow:

* [ ] Read-only
* [ ] Mutating files
* [ ] Mutating Git/GitHub/GitLab
* [ ] Calling CI/CD
* [ ] Calling internal APIs
* [ ] Using secrets or tokens
* [ ] Other:

## 5. Current files/configs

Which files do you maintain today?

* [ ] `AGENTS.md`
* [ ] `CLAUDE.md`
* [ ] `SKILL.md`
* [ ] MCP config
* [ ] OpenCode config
* [ ] OpenCode commands
* [ ] OpenCode agents/subagents
* [ ] VS Code config
* [ ] Cursor rules
* [ ] Cline rules/skills
* [ ] Continue config
* [ ] Gateway policy
* [ ] Other:

Optional: paste a sanitized tree.

```text
.
├── AGENTS.md
├── opencode.jsonc
└── .opencode/
    ├── commands/
    ├── agents/
    └── skills/
```

## 6. Safety rules

What must be allowed, denied, mutated, approved, or audited?

Examples:

```text
- deny `.env` and `secrets/**`
- require user confirmation before PR creation
- force draft PR
- force branch prefix `gpt/`
- allow only selected repositories
- run tests before mutation
- audit tool-call inputs and results
```

## 7. Desired generated output

What generated artifact would make Actlane useful for you?

* [ ] `actlane.yaml`
* [ ] `AGENTS.md`
* [ ] `SKILL.md`
* [ ] MCP tool metadata/config
* [ ] OpenCode command
* [ ] OpenCode agent/subagent
* [ ] Codex skill/config
* [ ] VS Code MCP config
* [ ] OpenAPI Action
* [ ] Policy bundle
* [ ] `actlane.lock`
* [ ] Safe remove plan
* [ ] Other:

## 8. Success criteria

Complete this sentence:

```text
Actlane would be useful to me if it could...
```
