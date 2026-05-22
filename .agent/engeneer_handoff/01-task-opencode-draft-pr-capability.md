# Task: OpenCode Draft PR Capability MVP

## Context

Actlane MVP must prove one narrow idea:

```text
one Capability contract
-> deterministic OpenCode artifacts
-> reproducible lockfile
-> CI-friendly validation
```

This task intentionally excludes MCP generation, runtime enforcement, gateway integration, registry, marketplace, brownfield apply/remove, and multi-target generation.

The first capability is a GitHub draft pull request workflow for SRE, MLOps, and developer teams.

## Goal

Create a minimal Actlane capability pack that generates only an OpenCode contract for safely preparing a GitHub draft PR.

The generated OpenCode output must be easy to use locally and easy to verify in CI/CD.

## 1. Generated Folder Structure For Capabilities

Decision: keep source manifests separate from generated target artifacts.

Canonical pack layout:

```text
packs/create-github-draft-pr/
  actlane.yaml
  capabilities/
    create-github-draft-pr.yaml
  policies/
    github-draft-pr.policy.yaml
  target-profiles/
    opencode.yaml
  generated/
    opencode/
      opencode.snippet.jsonc
      commands/
        create-github-draft-pr.md
      agents/
        github-draft-pr.md
      instructions/
        github-draft-pr.md
    policies/
      policy-bundle.json
    actlane.lock
```

Generated project-adoption layout for later brownfield use:

```text
.actlane/
  generated/
    opencode/
      opencode.snippet.jsonc
      commands/
        create-github-draft-pr.md
      agents/
        github-draft-pr.md
      instructions/
        github-draft-pr.md
  state/
    ownership.json
  reports/
    generate.json
```

OpenCode install target, when explicit apply exists later:

```text
.opencode/
  commands/
    create-github-draft-pr.md
  agents/
    github-draft-pr.md
opencode.jsonc
```

For the MVP, `actlane generate` writes only to `generated/` or `.actlane/generated/`. It must not edit `.opencode/` or `opencode.jsonc`.

### Acceptance Criteria

- `capabilities/*.yaml` is the only source location for capability contracts.
- Generated OpenCode files are under `generated/opencode/` by default.
- No command writes to `.opencode/` in MVP.
- Generated output can be deleted and recreated deterministically.
- Generated files include enough metadata to identify pack, capability, target, and generator version.

## 2. Minimal Capability Schema

Decision: the first schema is strict enough for useful generation, but not a workflow language.

Minimal `Capability` document:

```yaml
$schema: https://actlane.ru/schemas/v1alpha1/capability.schema.json
apiVersion: actlane.ru/v1alpha1
kind: Capability
metadata:
  name: create-github-draft-pr
  title: Create GitHub Draft PR
  description: Safely prepare a GitHub draft pull request from reviewed changes.
spec:
  whenToUse: Use after changes are reviewed and the user asks to prepare a GitHub draft PR.
  targets:
    - opencode
  inputs:
    repo:
      type: string
      required: true
    baseBranch:
      type: string
      required: true
    branch:
      type: string
      required: true
    title:
      type: string
      required: true
    summary:
      type: string
      required: true
    files:
      type: array
      items:
        type: string
      required: true
    confirmed:
      type: boolean
      required: true
  outputs:
    prUrl:
      type: string
    branch:
      type: string
    policyDecision:
      type: string
    summary:
      type: string
  policies:
    - github-draft-pr-policy
  toolFlow:
    - tool: git.status
      purpose: inspect changed files
    - tool: tests.run
      purpose: run project test command when configured
    - tool: github.create_branch
      purpose: create protected branch
    - tool: github.commit_files
      purpose: commit reviewed files
    - tool: github.create_pull_request
      purpose: create draft pull request
  reporting:
    includeChangedFiles: true
    includePolicyDecision: true
    includePrUrl: true
```

Minimum required schema fields:

```text
apiVersion
kind
metadata.name
spec.whenToUse
spec.targets
spec.inputs
spec.outputs
spec.policies
```

MVP validation rules:

- `apiVersion` must be `actlane.ru/v1alpha1`.
- `kind` must be `Capability`.
- `metadata.name` must be kebab-case.
- `spec.targets` must contain only `opencode` for this MVP.
- each policy reference must resolve to a policy file in the pack.
- unsupported fields are allowed only under `x-*` extension keys.

### Acceptance Criteria

- Invalid `apiVersion`, `kind`, missing `metadata.name`, or empty `spec.targets` fails validation.
- A capability targeting anything except `opencode` fails in this MVP.
- Policy references are resolved and missing policy names fail validation.
- Schema supports editor autocomplete through `$schema`.
- Schema remains small enough to explain in one README section.

## 3. Schema Hosting And Storage

Decision: use website-hosted JSON Schemas with stable URLs, following the same developer experience pattern used by OpenCode configs.

OpenCode uses a hosted schema URL such as:

```json
{
  "$schema": "https://opencode.ai/config.json"
}
```

Actlane should use:

```text
https://actlane.ru/schemas/v1alpha1/capability.schema.json
https://actlane.ru/schemas/v1alpha1/capability-pack.schema.json
https://actlane.ru/schemas/v1alpha1/tool-call-policy.schema.json
https://actlane.ru/schemas/v1alpha1/target-profile.schema.json
https://actlane.ru/schemas/v1alpha1/adoption-profile.schema.json
```

Repository source of truth:

```text
spec/v1alpha1/schemas/
  capability.schema.json
  capability-pack.schema.json
  tool-call-policy.schema.json
  target-profile.schema.json
  adoption-profile.schema.json
```

Published website path:

```text
https://actlane.ru/schemas/v1alpha1/
```

Publishing rule:

```text
spec/v1alpha1/schemas/*.schema.json
-> copied to the website deployment outside this repository
-> served with application/schema+json or application/json
```

Schema compatibility rule:

- patch changes may clarify descriptions or add optional fields;
- breaking changes require a new version path, for example `v1alpha2`;
- old schema URLs must remain available.

### Acceptance Criteria

- Every YAML example includes a `$schema` URL.
- Local schemas and published schemas are byte-identical in CI.
- `https://actlane.ru/schemas/v1alpha1/adoption-profile.schema.json` is reserved even if adoption is not MVP.
- Schema URLs are versioned by API version.
- The CLI can validate using local embedded schemas without requiring network access.

## 4. Development Framework

Decision: Actlane implementation will be written in Go.

Initial Go boundaries:

```text
packages/cli/
  go.mod
  cmd/actlane/
    main.go
  internal/cli/
  internal/schema/
  internal/pack/
  internal/capability/
  internal/policy/
  internal/generator/
  internal/generator/opencode/
  internal/lockfile/
  internal/fs/
```

Recommended Go libraries:

```text
CLI: github.com/spf13/cobra
YAML: gopkg.in/yaml.v3
JSON Schema: github.com/santhosh-tekuri/jsonschema/v6
Diff/check output: standard library first
Testing: Go testing package + golden files
```

Rules:

- generation must be deterministic;
- output maps must be ordered before rendering;
- no network access is required for validate or generate;
- generated files are written atomically;
- tests use golden fixtures for the OpenCode target.

### Acceptance Criteria

- Repository has a single Go module before implementation starts.
- `go test ./...` from `packages/cli` is the baseline CI command.
- CLI command parsing lives outside generator logic.
- OpenCode generation is isolated in its own adapter package.
- Validation and generation can be tested without shelling out to OpenCode or GitHub.

## 5. Minimal And Extensible CLI Commands

Decision: MVP CLI is generation-first and CI-friendly.

MVP commands:

```bash
actlane version
actlane validate <pack>
actlane generate <pack> --target opencode
actlane generate <pack> --target opencode --out .actlane/generated
actlane generate <pack> --target opencode --check
actlane generate <pack> --target opencode --frozen-lockfile
actlane schema list
actlane schema print capability
```

Command behavior:

```text
validate
  loads pack manifest, capabilities, policies, target profile, schemas
  returns non-zero on invalid input

generate
  validates first
  writes deterministic OpenCode artifacts
  writes actlane.lock

generate --check
  CI mode
  does not write files
  fails if generated output is missing or stale

generate --frozen-lockfile
  fails if lockfile is missing or would change

schema list
  prints available embedded schema names and URLs

schema print capability
  prints embedded capability.schema.json
```

Reserved future commands:

```bash
actlane inspect
actlane init --no-touch
actlane diff
actlane plan apply
actlane remove --dry-run
actlane check tool-call.json --policy generated/policies/policy-bundle.json
actlane pack create
actlane pack inspect
actlane pack install
actlane pack verify
```

### Acceptance Criteria

- `actlane validate <pack>` works without generating files.
- `actlane generate <pack> --target opencode` writes only generated output and lockfile.
- `actlane generate <pack> --target opencode --check` is safe for CI and does not write files.
- `actlane generate <pack> --target opencode --frozen-lockfile` detects drift.
- Unsupported targets fail with a clear message listing supported MVP targets.

## 6. Target Capability: GitHub Draft PR For OpenCode

Decision: the first generated runtime contract is OpenCode only.

Generated OpenCode snippet:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "instructions": [
    ".actlane/generated/opencode/instructions/github-draft-pr.md"
  ],
  "permission": {
    "edit": "ask",
    "bash": "ask"
  },
  "agent": {
    "github-draft-pr": {
      "description": "Prepare safe GitHub draft pull requests from reviewed changes.",
      "prompt": "Follow the generated github-draft-pr instructions and create only draft pull requests after explicit confirmation.",
      "tools": {
        "write": true,
        "edit": true
      }
    }
  },
  "command": {
    "create-github-draft-pr": {
      "description": "Prepare a safe GitHub draft PR",
      "agent": "github-draft-pr",
      "template": "Prepare a GitHub draft PR using the generated github-draft-pr capability contract. Refuse secrets, force draft=true, require confirmation before mutation, and report policy decision."
    }
  }
}
```

Generated markdown files are copy-ready for OpenCode file-based configuration:

```text
.actlane/generated/opencode/agents/github-draft-pr.md
  -> can be copied to .opencode/agents/github-draft-pr.md

.actlane/generated/opencode/commands/create-github-draft-pr.md
  -> can be copied to .opencode/commands/create-github-draft-pr.md
```

Generated command behavior:

```text
1. Inspect git status and changed files.
2. Refuse secret, credential, token, and private-key paths.
3. Refuse unreviewed or excessive file sets.
4. Run configured test command when present.
5. Require explicit user confirmation before mutation.
6. Create branch with required prefix, for example actlane/.
7. Commit reviewed files only.
8. Create GitHub PR as draft.
9. Return PR URL, branch, files, test status, and policy decision.
```

CI/CD usage:

```bash
actlane validate packs/create-github-draft-pr
actlane generate packs/create-github-draft-pr --target opencode --check
```

This lets SRE, MLOps, and developers verify that committed OpenCode capability artifacts are in sync with the source contract.

### Acceptance Criteria

- The pack contains exactly one MVP capability: `create-github-draft-pr`.
- Generated OpenCode artifacts do not require MCP.
- Generated command always creates draft PRs, never ready-for-review PRs.
- Branches are forced to the configured prefix.
- Secret-like paths are denied.
- Mutating GitHub actions require explicit confirmation.
- CI can validate schemas and stale generated output without GitHub credentials.

## Definition Of Done

- A developer can understand the whole MVP from this handoff and the generated files.
- The source capability contract is small and reviewable.
- The OpenCode target is useful without implementing runtime enforcement.
- The CLI surface is narrow but leaves room for future targets.
- The schema URL strategy is compatible with editor autocomplete and offline CLI validation.
