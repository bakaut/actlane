# SkillContract

`SkillContract` describes reusable skill semantics once and lets profile generators translate them into target-specific agent files such as Codex or OpenCode `SKILL.md`.

The contract is the source of truth for skill front matter, activation rules, security-gate flow, workflow steps, required inputs, and MCP tool references. Generators should not embed capability-specific prose in code.
