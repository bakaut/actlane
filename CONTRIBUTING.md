# Contributing To Actlane

Actlane is currently an RFC-stage repository. The most useful contributions are clear examples, target requests, spec feedback, and corrections to confusing documentation.

## Good First Contributions

- Explain a real agent workflow you want to make portable.
- Request a target runtime: Codex, OpenCode, VS Code, Cursor, Cline, Continue, Custom GPT, or Orchestra.
- Open a `Try Actlane on my setup` issue with your current files and desired generated artifact.
- Improve the `safe-gitops` example.
- Add policy examples for allow, deny, mutate, and requires-approval.
- Point out where the docs overpromise implementation that does not exist yet.

## Contribution Rules

- Keep the project generation-first.
- Keep runtime enforcement optional.
- Do not position Actlane as an MCP gateway.
- Do not add marketplace, SaaS, or enterprise scope to early phases.
- Prefer one concrete workflow over broad abstractions.
- Do not add commands to docs unless they exist or are clearly marked as planned.

## Phase 0 Documentation Style

Phase 0 documents should be:

- explicit about status;
- concrete about the first workflow;
- conservative about future capabilities;
- easy to read without knowing the whole repository;
- careful about safety claims.

## Development Status

There is no production CLI yet. Implementation contributions should wait until the Phase 0 examples and pack shape are clear enough to generate from.
