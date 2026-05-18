# Pack Format

An Actlane pack is a portable bundle of capability contracts, policies, generated artifacts, examples, and reproducibility metadata.

The first pack is:

```text
safe-gitops
```

## Intended Layout

```text
packs/safe-gitops/
  README.md
  actlane.yaml
  capabilities/
    create-safe-draft-pr.yaml
  policies/
    safe-gitops-pr.policy.yaml
  skills/
    create-safe-draft-pr/
      SKILL.md
  mcp/
    tools/
      create-safe-draft-pr.tool.json
  adapters/
    opencode/
    vscode/
    codex/
    custom-gpt/
  generated/
    agents/
    mcp/
    openapi/
    policies/
    tests/
  examples/
    tool-calls/
    outputs/
  pack.lock
  checksums.txt
```

## Source Files

The source of truth should be:

```text
actlane.yaml
capabilities/*.yaml
policies/*.yaml
target-profiles/*.yaml
```

Generated artifacts should be derived from those files.

## Generated Files

Generated output should be clearly separated:

```text
generated/
.actlane/generated/
```

Generated files should include metadata or ownership markers so they can be reviewed, updated, and removed safely.

## Lockfile

The lockfile records exact generated state:

```text
pack version
spec version
generator version
adapter versions
capability digests
policy digests
generated files
owned blocks
checksums
applied targets
audit metadata
```
