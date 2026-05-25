# ResponsibilityContract

`ResponsibilityContract` describes the verifiable responsibility boundary between a human, an AI agent, a repository, tools, and CI.

It is the Actlane source of truth for:

- instruction sources and precedence;
- repository scopes and risk floors;
- risk classes and required checks;
- shell, filesystem, git, and MCP permissions;
- human approval boundaries;
- evidence and handoff requirements;
- audit expectations.

It does not describe CLI UX, pack transport, generated file paths, or Go implementation details.

## Minimal Shape

```yaml
apiVersion: actlane.ru/v1alpha1
kind: ResponsibilityContract
metadata:
  name: default
  version: 0.1.0-alpha.1
spec:
  sources: {}
  precedence: {}
  scopes: []
  risk: {}
  checks: {}
  permissions: {}
  tools: {}
  humanBoundary: {}
  evidence: {}
  agentBehavior: {}
  handoffFormat: {}
  ci: {}
  audit: {}
```

## Projection Rule

If a target runtime has a native primitive for part of the contract, Actlane may generate that primitive.

If a target runtime does not have a native primitive, Actlane keeps the rule in the responsibility contract and exposes it through generated instructions, audit output, or the optional Actlane contract MCP interface.

## Non-Goals

`ResponsibilityContract` must not become:

- a replacement for `Capability`;
- a replacement for `SkillContract`, `CommandContract`, or `AgentContract`;
- a replacement for MCP runtime permissions;
- an MCP gateway;
- a production security guarantee.
