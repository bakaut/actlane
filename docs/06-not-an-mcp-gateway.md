# Actlane Is Not An MCP Gateway

MCP is a protocol for exposing tools, resources, and prompts.

Actlane is a contract layer for safe reusable agent capabilities.

## Difference

```text
MCP = how an agent reaches tools
Actlane = what a safe action means and which policy it obeys
```

An MCP gateway may route and protect traffic. Actlane defines the capability contract and can generate metadata or policy bundles that a gateway may use later.

## Why This Matters

If Actlane starts as a gateway, the project becomes too broad too early:

- auth;
- routing;
- latency;
- deployment;
- tenancy;
- logs;
- secrets;
- enterprise policy.

Those are real problems, but they are not the first MVP.

## Correct MVP

The first useful proof is:

```text
one safe-gitops pack
one create-safe-draft-pr capability
one actlane.yaml
generated artifacts
policy bundle
allow / deny / mutate examples
```

Runtime enforcement can come later.
