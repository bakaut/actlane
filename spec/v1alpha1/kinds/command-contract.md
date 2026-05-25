# CommandContract

`CommandContract` describes a user-facing command entrypoint for an Actlane capability.

The contract owns invocation metadata, argument handling, references to capability/skill/agent, and thin prompt text for a command file.

Capability semantics, output contracts, safety rules, approval policy, MCP bindings, runtime execution, and target-specific generated paths remain in their own contracts.
