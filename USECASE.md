# When to Use Actlane

Use Actlane when a simple `SKILL.md`, command, agent profile, or MCP config is no longer enough.

Actlane is useful when an AI-agent action must be:

* reusable across different agent runtimes;
* safe for mutating operations;
* connected to real MCP tools;
* governed by explicit policy;
* generated into agent-specific files;
* reproducible through a lockfile;
* removable without breaking the project.

---

## Core Use Cases

### 1. Safe mutating agent actions

Use Actlane when an agent can change real systems:

* create branches;
* commit files;
* open pull requests;
* trigger CI/CD;
* update issues;
* write to memory;
* call internal APIs.

Example:

```text
create-github-draft-pr
```

Instead of exposing raw tools:

```text
github.create_branch
github.push_files
github.create_pull_request
```

Actlane exposes one high-level capability:

```text
create-github-draft-pr
```

with policy, schema, skill instructions, MCP binding, generated agent files, and lockfile.

---

### 2. One capability, many agent runtimes

Use Actlane when the same workflow must work in different environments:

* OpenCode;
* Codex;
* Claude Code;
* VS Code Copilot;
* Cursor;
* Cline;
* Continue;
* Custom GPT.

Actlane lets you define the capability once and generate runtime-specific projections:

```text
actlane.yaml
→ AGENTS.md
→ SKILL.md
→ MCP metadata
→ OpenAPI Action
→ OpenCode command
→ VS Code MCP config
→ policy bundle
→ actlane.lock
```

---

### 3. Private capability packs

Use Actlane when teams need to share safe agent workflows privately.

Example packs:

```text
safe-gitops
safe-github-pr
safe-ci-debug
safe-docs-update
safe-jira-triage
```

A pack can contain:

```text
capabilities
policies
skills
MCP bindings
agent instructions
target profiles
generated adapters
lockfile
```

This makes agent workflows portable between developers without forcing everyone to use the same agent runtime.

---

### 4. Policy-aware skills

Use Actlane when a normal skill is too weak.

A primitive `SKILL.md` says:

```text
Agent, do this workflow.
```

Actlane says:

```text
Here is the capability.
Here is when to use it.
Here is the input/output schema.
Here are the allowed tools.
Here is the policy.
Here are the generated files.
Here is how to reproduce or remove it.
```

---

### 5. MCP tool safety

Use Actlane when raw MCP tools are too low-level or risky.

MCP gives the agent tools.

Actlane turns raw tools into safe capabilities:

```text
raw MCP tools
→ high-level capability
→ policy
→ generated skill
→ generated command
→ optional runtime enforcement
```

---

### 6. Brownfield adoption

Use Actlane when you want to add agent capabilities to an existing project without breaking it.

Actlane should follow this lifecycle:

```text
inspect → init → generate → plan → apply → remove
```

By default:

* generate beside the project;
* show diffs before changes;
* never overwrite unknown files;
* track owned files and blocks;
* remove only Actlane-owned artifacts.

---

## When Not to Use Actlane

Actlane may be unnecessary if you have:

* one agent;
* one repository;
* one simple readonly workflow;
* no mutating tools;
* no need to share packs;
* no need for policy, lockfile, or generated adapters.

In that case, a simple `AGENTS.md` or `SKILL.md` may be enough.

---

## Why Actlane Is Different

| Primitive      | What it does                | What is missing                                |
| -------------- | --------------------------- | ---------------------------------------------- |
| `SKILL.md`     | Teaches an agent a workflow | No strict schema, policy, lockfile             |
| Command        | Gives a shortcut            | Usually just a prompt wrapper                  |
| Agent/subagent | Defines a role              | Runtime-specific                               |
| MCP tool       | Exposes a function          | Too low-level by itself                        |
| MCP gateway    | Routes and controls traffic | Does not define portable capability meaning    |
| Actlane pack   | Ties everything together    | Capability contract, policy, targets, lockfile |

---

## Value

Actlane provides:

```text
portable capability contracts
policy-aware agent actions
generated runtime adapters
MCP binding visibility
safe brownfield adoption
reproducible generation
safe remove
optional enforcement
```

---

## One-Line Summary

Actlane packages safe AI-agent actions as portable, policy-aware capability packs that can generate `AGENTS.md`, `SKILL.md`, MCP metadata, agent configs, policy bundles, and lockfiles.

```text
Primitive skills teach the agent.
Actlane packs govern, package, generate, and reproduce agent capabilities.
```
