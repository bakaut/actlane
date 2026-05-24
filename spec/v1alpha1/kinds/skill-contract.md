# SkillContract

`SkillContract` describes the portable skill folder contract used by agent profiles such as Claude, Codex, and OpenCode.

The contract has only the primitives needed to form a target skill directory:

- `metadata.name` and `metadata.description` for `SKILL.md` front matter.
- `spec.body` for `SKILL.md` content.
- optional `spec.scripts`, `spec.references`, and `spec.assets` resources copied under the matching skill subdirectories.

Runtime policy, MCP bindings, capability inputs, and tool execution remain in their own Actlane contracts.
