# Responsibility Contract

`ResponsibilityContract` describes the responsibility boundary around a capability pack.

It is the policy source for risk, permissions, human approval, required checks, evidence, and audit expectations. It is not a CLI workflow and it is not a generated agent artifact.

## Why It Exists

Agent instructions, MCP configs, CI rules, and PR templates all express parts of the same safety model. Without a shared contract, those rules drift across prompts, runtime config, shell commands, and human memory.

`ResponsibilityContract` keeps that model reviewable:

```text
human boundary
repository scopes
risk classes
required checks
tool permissions
evidence rules
audit expectations
```

## Boundary

The contract answers:

```text
Is this action allowed?
How risky is it?
Which checks are required?
Is human approval required?
What evidence must be returned?
Which tools and paths are governed?
```

The CLI answers:

```text
Where is the contract?
Is it valid?
How is it packed?
How is it installed?
How is it projected into a target profile?
How is generated state locked or reported?
How is adoption, apply, and remove performed?
```

These responsibilities should stay separate. The YAML contract must not describe CLI command UX, zip packaging mechanics, Go generator internals, target-specific implementation details, or apply/remove lifecycle.

## Relation To Other Kinds

`Capability` describes the safe user action.

`SkillContract` describes a portable skill directory: `SKILL.md` plus optional scripts, references, and assets.

`CommandContract` describes a user entrypoint for an agent runtime.

`AgentContract` describes a narrow agent role.

`MCPBinding` describes real tool wiring.

`ToolCallPolicy` describes specific allow, deny, mutate, and approval behavior for tool calls.

`ResponsibilityContract` ties the safety expectations around those pieces without replacing them.

## Static Projection

Actlane may project parts of the contract into files that agents can read:

```text
AGENTS.md
SKILL.md
commands
agent profiles
PR templates
runtime config comments or snippets
```

Static projection should explain:

```text
what to check before mutation
when to ask a human
which files and tools are sensitive
which verification evidence is required
what must not be claimed without proof
```

## Active Contract Interface

For MVP runtime use, Actlane can expose a small contract oracle through MCP or CLI.

It should not proxy GitHub, replace runtime permissions, or become a general MCP gateway.

Minimum interface:

```text
resources:
  actlane://contract
  actlane://handoff-format

tools:
  actlane.check_action
  actlane.required_evidence
  actlane.record_evidence
```

The key check is:

```text
changed files -> scopes -> risk floor -> required checks -> approval / stop decision
```

If the target runtime has a native primitive, Actlane can generate it. If not, the rule stays Actlane-native and is surfaced through generated instructions plus the contract oracle.

## MVP Acceptance

For `packs/create-github-draft-pr`, the first useful cases are:

```text
docs change -> low risk + lint
package code change -> medium risk + lint + unit tests
CI workflow change -> high risk + human approval
secret path change -> critical risk + stop
unregistered MCP write tool -> policy violation
missing PR evidence section -> audit failure
```

This keeps Actlane focused on a thin vertical slice: one pack, one capability, one responsibility contract, static projection, and a small active check surface.
