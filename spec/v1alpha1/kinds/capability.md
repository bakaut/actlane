# Capability

`Capability` describes one high-level agent action.

It should expose a user-meaningful workflow instead of raw tool calls.

Example:

```text
create-safe-draft-pr
```

## Required Ideas

- when the capability should be used;
- input schema;
- output schema;
- policy references;
- underlying tool mapping;
- generated target artifacts;
- reporting contract for the agent.

## Phase 0 Boundary

This kind is a contract sketch. No CLI validation is implemented yet.
