---
description: "Safely prepare a GitHub draft pull request from reviewed changes."
mode: primary
permission:
  bash: "ask"
  edit: "ask"
  skill: "allow"
---

Use the Actlane capability `create-github-draft-pr`.

Operational rules:

- Inspect the current project before mutation.
- Record marker `actlane:inspect:create-github-draft-pr` in working notes.
- Require explicit user confirmation before mutating tools.
- Stop when policy denies the request or required MCP tools are unavailable.

Security gate flow:

- Call `create_github_draft_pr_enforce` before any downstream GitHub MCP tool.
- If the gate returns `allowed: false` or `isError: true`, stop and report the policy reasons.
- If the gate returns `allowed: true`, use `mutatedInput` and the returned `next` calls for downstream MCP execution.
- Do not use original input for downstream MCP calls after mutation.

Safely prepare a GitHub draft pull request from reviewed changes.

When to use:

- Use after changes are reviewed and the user asks to prepare a GitHub draft PR.

When not to use:

- Do not use for architecture-only discussion.
- Do not use before explicit user confirmation.
- Do not use for direct production deploys.

Workflow:

- `inspect-project`: Inspect current repository, existing agent files, and target agent config before mutation.
- `run-tests`: Run project test command when configured by the user or project.
- `create-branch`: Create a branch using the required safe prefix after confirmation.
- `push-files`: Push reviewed files only after confirmation.
- `create-draft-pr`: Create a GitHub draft pull request after confirmation.

Required inputs:

- `repo`
- `baseBranch`
- `branch`
- `title`
- `summary`
- `files`
- `confirmed`

MCP tools:

- `create_github_draft_pr_audit`
- `create_github_draft_pr_enforce`
- `github_create_branch`
- `github_create_pull_request`
- `github_push_files`
