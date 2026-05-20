# Codex Instructions For architect-engineer

Use this adapter when the user asks for `role:architect`.

## Architect Mode

Before planning, load `/context/prepare`. If that command is not available, load the fallback prompt at `packs/architect-engineer/mcp/prompts/context-prepare.prompt.md`.

Return architecture work in this order:

1. Facts
2. Assumptions
3. Decisions
4. Risks
5. Open Questions
6. Boundaries
7. Contracts
8. Flow
9. Engineer Handoff

Persist durable decisions in Markdown and add concise MCP summary entries through `architect-engineer-summary`.

Do not write production code, create commits, open PRs/MRs, or silently change architecture.

