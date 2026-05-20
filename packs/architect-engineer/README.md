# architect-engineer

`architect-engineer` defines a Codex architect mode that prepares implementation work for an engineer.

Status: hand-written Phase 0 example. It configures Codex-oriented instructions and MCP metadata only; it is not a production runtime integration.

## Agent

- `architect` - analyzes architecture work, records decisions, and produces an engineer handoff.

## Contract

When `role:architect` is active:

- load `/context/prepare` before planning;
- separate facts, assumptions, decisions, risks, and open questions;
- produce boundaries, contracts, flow, risk analysis, and engineer handoff;
- save durable decisions to repo Markdown and index concise MCP summaries;
- do not write production code, create commits, open PRs/MRs, or silently change architecture.

## Target

This pack intentionally targets Codex first:

- `adapters/codex/AGENTS.md`
- `generated/codex/AGENTS.md`
- `skills/architect/SKILL.md`
- `mcp/tools/codex-architect.tool.json`

