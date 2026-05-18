# Actlane Manifesto

## MCP gives agents hands. Actlane defines the safe lane for every action.

AI agents are no longer just chat interfaces. They read repositories, call tools, run tests, create pull requests, write to memory, interact with CI/CD, call internal APIs, and increasingly gain the ability to change the real world.

But the rules for these actions are scattered across too many places:

```text
prompt
AGENTS.md
SKILL.md
MCP tool schema
CLI/Codex/Claude/OpenCode config
opencode.json
.claude/mcp.json
gateway policy
CI scripts
README
an engineer's memory
```

The same capability lives in ten different formats. The same safety rule is copy-pasted by hand. The same agent workflow is interpreted differently by every runtime.

For read-only tasks, this is inconvenient. For mutating tools, it is dangerous.

## The problem

You cannot simply give an AI agent a bag of tools and hope that a prompt will keep it safe.

A prompt can say:

```text
be careful
do not touch secrets
only create draft pull requests
run tests first
```

But a prompt is not enforcement.

The real problem is this:

```text
How do we safely, portably, and reproducibly describe
what an agent is allowed to do,
when it should use an action,
which inputs are valid,
which safe defaults must be applied,
what artifacts should be generated,
and where the same contract can be enforced at runtime?
```

## What Actlane is

**Actlane** is a DSL-first system for describing portable, policy-aware capabilities for AI agents.

In short:

```text
Actlane turns scattered agent prompts, tools, configs, and policies
into one typed, reusable, and auditable capability contract.
```

Actlane is not an agent. Actlane is not an MCP gateway. Actlane is not another IDE or runtime framework. Actlane is not a skill marketplace.

Actlane is a **source of truth** for safe agent actions.

## The core idea

```text
Define safe agent actions once.
Generate agent/runtime integrations everywhere.
Optionally enforce the same contract at runtime.
```

One `actlane.yaml` can generate:

```text
AGENTS.md
SKILL.md
MCP tool metadata
MCP prompts/resources
CLI command docs
Codex instructions
Claude instructions
OpenCode commands/config
policy bundle
contract tests
audit schema
```

## Example

Instead of giving an agent low-level tools like:

```text
github.create_branch
github.commit_files
github.create_pull_request
sandbox.run_tests
```

Actlane describes a high-level capability:

```text
create-safe-draft-pr
```

Inside that capability, the contract defines:

```text
when to use it
what to check before calling it
input/output schemas
forbidden paths
required branch prefix
whether the PR must be a draft
whether tests should run first
which tools are called underneath
how the agent should report the result
what must be audited
```

## The value

### For developers

Less manual copy-paste between agent runtimes.

```text
one capability contract
→ many generated adapters
```

A developer can pass a pack to another developer even if they use a different environment: CLI agents, Codex, Claude, or OpenCode.

### For teams

A single place to describe:

```text
what an agent may do
what is forbidden
which safe defaults apply
how to connect a capability to different tools
how to remove or roll back an integration
```

### For security, platform, and SRE teams

The prompt stops being the only safety barrier.

Actlane can describe:

```text
validation rules
safe mutations/defaults
approval hints
audit requirements
runtime policy bundles
```

You do not need to start with a runtime. First, Actlane can simply generate artifacts. Later, the same contract can be enforced through a webhook, sidecar, MCP wrapper, or an existing gateway.

## Reproducibility: `actlane.lock`

Actlane needs the equivalent of a lockfile.

```text
actlane.yaml = desired capability contract
actlane.lock = exact generated and applied state
```

Like `package-lock.json` makes dependency resolution reproducible, `actlane.lock` makes agent capability generation reproducible.

It can record:

```text
pack version
spec version
generator version
adapter versions
capability digests
policy digests
generated files
owned blocks
checksums
applied targets
audit metadata
```

This gives teams:

```text
reproducible generation
drift detection
reviewable capability changes
safe rollback
safe remove
supply-chain visibility
```

Without a lockfile, a regenerated `SKILL.md`, MCP tool, CLI/Codex/Claude/OpenCode config, or agent config may silently change after a generator, template, or adapter update.

With `actlane.lock`, teams can run:

```text
actlane generate --frozen-lockfile
actlane verify-lock
actlane diff-lock
actlane remove --from-lock
```

The lockfile makes Actlane boring in the best possible way: predictable, reviewable, and safe to remove.

## A key principle

Actlane is designed for safe adoption.

```text
inspect → init → generate → plan → apply → remove
```

It does not break an existing project. It does not overwrite files it does not own. It does not become the owner of your repository.

By default:

```text
generate beside the project
show the diff
apply only explicitly
remove only owned files and owned blocks
```

Install should be boring. Apply should be explicit. Remove should be trustworthy.

## What Actlane is not

Actlane does not replace MCP.

```text
MCP = protocol for tools/resources/prompts.
Actlane = contract for safe reusable agent capabilities.
```

Actlane does not replace `AGENTS.md`.

```text
AGENTS.md = project-level guidance.
Actlane = source of truth that can generate or update guidance.
```

Actlane does not replace `SKILL.md`.

```text
SKILL.md = reusable workflow/instruction package.
Actlane = generator and policy layer for such skills.
```

Actlane does not replace an MCP gateway.

```text
A gateway routes and protects traffic.
Actlane defines what each tool-call means and which policy it must obey.
```

Actlane does not replace an agent framework.

```text
An agent decides what to do next.
Actlane defines safe lanes for what it is allowed to do.
```

## The first practical focus

We are not starting with an enterprise SaaS. We are not starting with a marketplace. We are not starting with our own gateway as the main product.

The first focus is:

```text
portable private capability packs
```

The first pack:

```text
safe-gitops
```

The first workflow:

```text
safe AI-assisted draft pull request creation
```

The minimal demonstration:

```text
one actlane.yaml
→ AGENTS.md
→ SKILL.md
→ MCP metadata
→ CLI / Codex / Claude / OpenCode snippets
→ policy bundle
→ examples of allow / deny / mutate tool-calls
```

## Why now

The AI-agent ecosystem is fragmenting fast:

```text
different IDEs
different MCP configs
different skill formats
different prompt conventions
different gateway policies
different agent runtimes
```

But all teams need the same thing:

```text
a safe way to move capabilities, context, and rules
between people, teams, and agent environments
```

Actlane lives exactly there:

```text
between prompts and tools
between MCP and gateways
between private infrastructure and portable skills
between developer convenience and security enforcement
```

## One sentence

```text
Actlane packages safe AI-agent actions as portable, policy-aware capability packs that can generate AGENTS.md, SKILL.md, MCP metadata, CLI/Codex/Claude/OpenCode configs, agent configs, policy bundles, and optional runtime enforcement artifacts.
```

## Motto

```text
MCP connects agents to tools.
Actlane defines the safe lane for every agent action.
```

Or, more directly:

```text
Stop hiding safety rules in prompts.
Make agent actions typed, portable, and enforceable.
```
