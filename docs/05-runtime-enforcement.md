# Runtime Enforcement

Runtime enforcement is optional.

Actlane should first prove generation mode:

```text
actlane.yaml
-> generated artifacts
-> policy bundle
-> actlane.lock
```

Only after that should runtime enforcement be added.

## Possible Runtime Modes

Potential later integrations:

```text
generic webhook
local sidecar
MCP wrapper
gateway plugin
native gateway policy
```

## Policy Decisions

A generated policy bundle should eventually support:

```text
allow
deny
mutate
requires_approval
```

Example:

```text
create-safe-draft-pr
  input branch = "feature/foo"
  policy mutates branch = "actlane/feature/foo"
  policy forces draft = true
```

## Guardrail

Actlane should not require a daemon or gateway for its core value.

Correct positioning:

```text
Bring your own gateway.
Use Actlane as the capability contract and policy source of truth.
```
