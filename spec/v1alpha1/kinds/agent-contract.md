# AgentContract

`AgentContract` describes a role boundary for a target AI agent profile.

The contract owns agent identity, mode, role summary, activation hints, allowed capabilities, allowed skills, and tool strategy.

Global permissions, evidence requirements, capability semantics, policies, MCP servers, command invocation, and target-specific generated paths remain in their own contracts.

Subagents are represented by the same `AgentContract` kind with `spec.mode: subagent`.
Actlane does not introduce a separate `SubagentContract`; target profiles decide how that agent is projected into their native subagent or agent file layout.
