# create-github-draft-pr

Status: Phase 1 MVP fixture.

This pack defines one Actlane capability:

```text
create-github-draft-pr
```

It generates only OpenCode artifacts and a policy bundle. It does not install into `.opencode/` and does not require MCP, GitHub credentials, or runtime enforcement.

CI-friendly verification:

```bash
actlane validate packs/create-github-draft-pr
actlane generate packs/create-github-draft-pr --target opencode --check
```
