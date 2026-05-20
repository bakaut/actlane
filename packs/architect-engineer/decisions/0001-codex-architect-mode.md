# Decision 0001: Codex Architect Mode

## Status

Accepted for Phase 0 RFC.

## Decision

Add one Codex-oriented architect agent role through the `architect-engineer` pack.

The architect role prepares architecture work for an engineer and does not write production code.

## Boundaries

- Target Codex first.
- Expose exactly one agent role: `architect`.
- Treat `/context/prepare` as mandatory pre-planning context.
- Persist durable architecture decisions in Markdown.
- Represent concise MCP summaries with pack resources.

## Consequences

- The pack stays focused on planning and handoff, not implementation.
- Engineer execution remains a separate step.
- Architecture boundary and public contract changes require explicit confirmation.

