# Actlane v1alpha1

`v1alpha1` is the first draft of the Actlane contract model.

It is designed to support the first no-code proof:

```text
safe-gitops
create-safe-draft-pr
generated AGENTS.md / SKILL.md / MCP metadata / CLI/Codex/Claude/OpenCode config / policy bundle
```

## Kinds

- [Capability](kinds/capability.md)
- [ToolCallPolicy](kinds/tool-call-policy.md)
- [TargetProfile](kinds/target-profile.md)
- [AdoptionProfile](kinds/adoption-profile.md)
- [RuntimeBinding](kinds/runtime-binding.md)
- [CapabilityPack](kinds/capability-pack.md)

## Schemas

Schemas are intentionally permissive in Phase 0. They document shape and catch obvious mistakes, not every future rule.

## Examples

See [examples](examples/) for minimal YAML documents.
