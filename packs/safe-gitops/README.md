# safe-gitops

`safe-gitops` is the first Actlane capability pack.

Status: hand-written Phase 0 example. It demonstrates the desired shape of a pack before the CLI exists.

## Workflows

- `create-safe-draft-pr` - prepare a draft pull request with safety rules.
- `run-project-tests` - run configured project tests as a supporting read/execute workflow.

## Safety Rules

- Draft PR by default.
- Branch prefix `actlane/`.
- Sensitive paths denied.
- Mutating actions require explicit user confirmation.
- Policy bundle records allow, deny, mutate, and approval behavior.

## User Action

Open an issue with your current agent setup and target runtime. We will use real setups to decide which target adapter to build first.
