# create-github-draft-pr

Status: Phase 1 MVP fixture.

This pack defines one Actlane capability:

```text
create-github-draft-pr
```

It generates OpenCode and Codex artifacts plus target-local policy bundles. It does not install into `.opencode/`, `.codex/`, or `~/.codex/config.toml`; generated files stay under `generated/<target>/` for review and explicit apply.

CI-friendly verification:

```bash
actlane validate packs/create-github-draft-pr
actlane generate packs/create-github-draft-pr --target opencode --check
actlane generate packs/create-github-draft-pr --target codex --check
```

Large skill instructions use `SkillContract.spec.bodySource`, while YAML keeps
the skill name, trigger description, and generation ownership. Short skills
may continue to use inline `spec.body`.
