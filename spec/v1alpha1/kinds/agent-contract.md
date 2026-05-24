# AgentContract

`AgentContract` describes a role boundary for a target AI agent profile.

The contract owns agent identity, mode, allowed capabilities and skills, permissions, tool strategy, and expected output requirements. Capability semantics, policies, MCP servers, and command invocation remain in their own contracts.

Subagents are represented by the same `AgentContract` kind with `spec.mode: subagent`.
Actlane does not introduce a separate `SubagentContract`; target profiles decide how that agent is projected into their native subagent or agent file layout.
