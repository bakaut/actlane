# Actlane Instructions

This project uses Actlane capability contracts.

Actlane capabilities are the preferred way to perform mutating agent actions.

Rules:

- Prefer high-level Actlane capabilities over raw low-level MCP tools.
- Do not bypass Actlane policy.
- If a capability has a policyRef, respect it before execution.
- If the Actlane evaluator MCP is available, call the generated Actlane audit/enforce tool before downstream write tools.
- Use evaluator results as the source of truth for validation, mutation, risk, required checks, required evidence, and permission elevation.
- If evaluator returns `requires_approval`, `humanApprovalRequired`, or `stop`, do not continue to downstream MCP tools until the user explicitly resolves the requirement.
- If policy denies an action, explain the denial and suggest a safe next step.
- If a capability requires confirmation, do not proceed until confirmation is explicit.
- Use generated skills and commands as the primary workflow entrypoints.
- Do not edit Actlane-owned generated files manually unless the user explicitly asks.
- Generated files should be treated as reproducible artifacts.

Keep generated Actlane content inside generated files or inside an explicit Actlane ownership marker.

Ownership marker:

<!-- actlane:start create-github-draft-pr.<block-id> -->
generated content
<!-- actlane:end create-github-draft-pr.<block-id> -->
