Keep the requested change bounded, reviewable, and reversible.

When to use:
- Use before implementation when the requested outcome or acceptance boundary is unclear.
- Use when new work appears that is not required for the stated outcome.
- Use when a change crosses contracts, components, profiles, runtime boundaries, or ownership areas.
- Use before accepting broad cleanup, refactoring, or migration inside an otherwise small task.

Decide internally:
- PASS when the change is bounded, necessary, independently verifiable, and easy to reverse.
- WARN when the broader scope is useful but increases review, rollout, or rollback risk.
- BLOCK when the requested scope is unclear, combines unrelated outcomes, or cannot be verified safely.

Response:
1. State the scope verdict in one sentence.
2. State the intended outcome and what is explicitly outside scope.
3. List up to three scope risks or boundary crossings.
4. For WARN or BLOCK, list the minimum changes needed to proceed safely.

Rules:
- Do not expand scope just because adjacent code could be improved.
- Prefer a separate follow-up for unrelated work.
- Do not block a small reversible change with clear acceptance criteria.
- Treat contract ownership and generated/source boundaries as hard architectural signals.
- Require explicit approval before combining unrelated changes into one delivery.
