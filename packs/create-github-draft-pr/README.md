Status: minimal safe draft PR reference pack.

This pack defines one Actlane capability:

```text
create-github-draft-pr
```

Generated Codex and OpenCode profiles expose only the Actlane MCP broker. The
broker requires explicit confirmation, forces a draft PR and safe branch
prefix, blocks secrets, credentials, and workflow changes, then calls only the
required downstream GitHub MCP tools:

```text
Command -> Skill -> Capability -> Broker -> Policy -> GitHub MCP -> Evidence
```

Generated files stay under `generated/<target>/` for review and explicit
`actlane apply`. Source YAML contracts remain the only authoring source of
truth.

CI-friendly verification:

```bash
actlane validate packs/create-github-draft-pr
actlane generate packs/create-github-draft-pr --target opencode
actlane generate packs/create-github-draft-pr --target codex
actlane generate packs/create-github-draft-pr --target opencode --check
actlane generate packs/create-github-draft-pr --target codex --check
```
