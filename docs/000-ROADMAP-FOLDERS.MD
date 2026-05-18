# Actlane Phase-by-Phase File Structure

## Purpose

This document describes how the Actlane repository should grow phase by phase without becoming too broad too early.

The rule:

```text
Start with a readable RFC repo.
Add implementation only when the previous phase has proven demand.
Keep generated artifacts separate from source-of-truth manifests.
```

---

# Repository Principles

## Source of truth

```text
actlane.yaml
capabilities/*.yaml
policies/*.yaml
target-profiles/*.yaml
```

## Generated artifacts

```text
generated/
.actlane/generated/
```

## Reproducibility state

```text
actlane.lock
```

## Safe adoption state

```text
.actlane/state/
.actlane/backups/
.actlane/reports/
```

## Runtime is optional

Runtime files should not appear in the core path until a later phase.

---

# Phase 0 — Public RFC / Demand Validation

## Goal

Publish the idea, problem, proposed spec, examples, and generated artifacts without claiming that a production CLI exists.

## Files

```text
actlane/
  README.md
  STATUS.md
  MANIFESTO.md
  ROADMAP.md
  CONTRIBUTING.md
  SECURITY.md
  LICENSE

  docs/
    00-problem.md
    01-concept.md
    02-architecture.md
    03-pack-format.md
    04-brownfield-adoption.md
    05-runtime-enforcement.md
    06-not-an-mcp-gateway.md
    07-not-an-agent-framework.md
    08-private-packs.md

  spec/
    README.md
    v1alpha1/
      README.md
      kinds/
        capability.md
        tool-call-policy.md
        target-profile.md
        adoption-profile.md
        runtime-binding.md
        capability-pack.md
      schemas/
        capability.schema.json
        tool-call-policy.schema.json
        target-profile.schema.json
        adoption-profile.schema.json
        runtime-binding.schema.json
        capability-pack.schema.json
      examples/
        capability.yaml
        tool-call-policy.yaml
        target-profile.yaml
        adoption-profile.yaml
        runtime-binding.yaml
        capability-pack.yaml

  packs/
    README.md
    safe-gitops/
      README.md
      actlane.yaml
      AGENTS.md
      capabilities/
        create-safe-draft-pr.yaml
        run-project-tests.yaml
      policies/
        safe-gitops-pr.policy.yaml
        block-secrets.policy.yaml
      skills/
        create-safe-draft-pr/
          SKILL.md
          references/
            policy.md
            examples.md
            pr-checklist.md
          assets/
            pr-template.md
      mcp/
        tools/
          create-safe-draft-pr.tool.json
          run-project-tests.tool.json
        prompts/
          create-safe-draft-pr.prompt.md
        resources/
          safe-gitops-policy.resource.md
        server.json
      adapters/
        opencode/
          opencode.snippet.json
          commands/
            create-safe-draft-pr.md
        vscode/
          mcp.snippet.json
        codex/
          AGENTS.md
        custom-gpt/
          openapi.yaml
          system-prompt.md
          action-catalog.md
      generated/
        README.md
        openapi/
          openapi.yaml
        mcp/
          tools.json
          server.json
        policies/
          policy-bundle.json
        agents/
          AGENTS.generated.md
          skills.generated.md
        tests/
          contract-tests.yaml
      examples/
        tool-calls/
          allow-create-safe-draft-pr.json
          deny-secret-edit.json
          mutate-branch-prefix.json
        outputs/
          allow-result.json
          deny-result.json
          mutate-result.json
      pack.lock
      checksums.txt

  examples/
    README.md
    01-minimal-capability/
      README.md
      actlane.yaml
      generated/
        mcp-tool.json
        AGENTS.md
    02-create-safe-draft-pr/
      README.md
      actlane.yaml
      capabilities/
        create-safe-draft-pr.yaml
      policies/
        safe-gitops-pr.policy.yaml
      generated/
        openapi.yaml
        mcp-tool.json
        opencode.json
        vscode-mcp.json
        policy-bundle.json

  diagrams/
    README.md
    plantuml/
      actlane-overview.puml
      capability-development-flow.puml
      pack-portability-flow.puml
      brownfield-adoption-flow.puml
      optional-runtime-enforcement.puml
    svg/
      actlane-overview.svg
      capability-development-flow.svg
      pack-portability-flow.svg
      brownfield-adoption-flow.svg
      optional-runtime-enforcement.svg

  assets/
    brand/
      logo.svg
      logo-dark.svg
      wordmark.svg
      favicon.svg
    images/
      hero-overview.png
      fragmented-agent-configs.png
      one-capability-many-targets.png
      safe-action-lane.png

  website/
    README.md
    articles/
      mcp-security.html
      ai-agent-governance.html
      tool-call-policy.html
      mcp-tool-poisoning.html
      openai-actions-vs-mcp.html
      private-agent-capability-packs.html

  packages/
    README.md
    cli/
      README.md
    generator/
      README.md
    policy/
      README.md
    runtime/
      README.md
    sdk/
      README.md
```

## Notes

At this stage, `packages/*` are placeholders only. They explain intended boundaries but contain no implementation code.

## Acceptance File Signal

Phase 0 is complete when the repository can be read as a product/spec proposal and the `safe-gitops` pack demonstrates the core idea without executable code.

---

# Phase 1 — Minimal CLI Prototype

## Goal

Make the examples executable through a tiny CLI.

## New Files

```text
actlane/
  packages/
    cli/
      README.md
      src-or-placeholder/
        main.*
        commands/
          validate.*
          generate.*
      tests-or-placeholder/
        validate.safe-gitops.*
        generate.safe-gitops.*

    generator/
      README.md
      src-or-placeholder/
        load-pack.*
        render-target.*
        lockfile.*
        targets/
          agents.*
          mcp.*
          policy.*

  tests/
    fixtures/
      safe-gitops/
        actlane.yaml
        capabilities/
        policies/
    expected/
      safe-gitops/
        AGENTS.md
        skills/
          create-safe-draft-pr/
            SKILL.md
        mcp/
          create-safe-draft-pr.tool.json
        policies/
          policy-bundle.json

  .github/
    workflows/
      validate-spec.yml
      validate-examples.yml
```

## Generated Files

```text
packs/safe-gitops/generated/
  agents/
    AGENTS.generated.md
    skills/
      create-safe-draft-pr/
        SKILL.md
  mcp/
    create-safe-draft-pr.tool.json
  policies/
    policy-bundle.json
  tests/
    contract-tests.yaml

packs/safe-gitops/actlane.lock
```

## Source-of-Truth Files

```text
packs/safe-gitops/actlane.yaml
packs/safe-gitops/capabilities/*.yaml
packs/safe-gitops/policies/*.yaml
```

## Commands Proven by Files

```bash
actlane validate packs/safe-gitops
actlane generate packs/safe-gitops --target agents
actlane generate packs/safe-gitops --target mcp
actlane generate packs/safe-gitops --target policy
actlane generate packs/safe-gitops --frozen-lockfile
```

## Acceptance File Signal

Phase 1 is complete when generated outputs can be deleted and recreated deterministically from source manifests and `actlane.lock`.

---

# Phase 2 — Brownfield-Safe Adoption

## Goal

Allow a user to try Actlane inside an existing project without damaging files.

## New Files

```text
actlane/
  examples/
    03-brownfield-adoption/
      README.md
      existing-project/
        AGENTS.md
        opencode.json
        .vscode/
          mcp.json
      expected/
        .actlane/
          captrakt-report-placeholder.md
          generated/
            snippets/
              AGENTS.snippet.md
              opencode.snippet.json
              vscode-mcp.snippet.json
          state/
            owned-files.json
            owned-blocks.json
          backups/
            README.md
      reports/
        inspect.md
        plan-apply.md
        remove-dry-run.md

  packages/
    cli/
      src-or-placeholder/
        commands/
          inspect.*
          init.*
          diff.*
          plan.*
          remove.*

    generator/
      src-or-placeholder/
        adoption/
          inspect-project.*
          ownership-markers.*
          snippet-writer.*
          backup-plan.*
          remove-plan.*
```

## Project-Local Files Created by Actlane

```text
.actlane/
  actlane.yaml
  generated/
    snippets/
      AGENTS.snippet.md
      opencode.snippet.json
      vscode-mcp.snippet.json
  state/
    owned-files.json
    owned-blocks.json
    applied-patches.json
    checksums.json
  backups/
  reports/
    inspect.md
    diff.md
    remove-dry-run.md
```

## Ownership Markers

If Actlane writes into an existing file, it must write only inside owned blocks:

```markdown
<!-- actlane:start create-safe-draft-pr -->
Generated content
<!-- actlane:end create-safe-draft-pr -->
```

## Commands Proven by Files

```bash
actlane inspect
actlane init --no-touch
actlane generate
actlane diff
actlane plan apply
actlane remove --dry-run
```

## Acceptance File Signal

Phase 2 is complete when a brownfield example shows that Actlane can inspect and generate snippets without modifying existing project files by default.

---

# Phase 3 — More Targets, Still Generation-First

## Goal

Prove portability across several agent runtimes while keeping the core DSL stable.

## New Files

```text
actlane/
  packages/
    generator/
      src-or-placeholder/
        targets/
          opencode.*
          vscode.*
          openapi.*
          codex.*
          cline.*
          continue.*
          cursor.*
          orchestra.*

  tests/
    fixtures/
      target-generation/
        actlane.yaml
    expected/
      opencode/
        opencode.snippet.json
        commands/
          create-safe-draft-pr.md
      vscode/
        mcp.snippet.json
      openapi/
        openapi.yaml
      codex/
        AGENTS.md
      cline/
        mcp-settings.snippet.json
      continue/
        config.snippet.yaml
      cursor/
        rules.md
      orchestra/
        pack-manifest.yaml
        SKILL.md

  docs/
    targets/
      README.md
      opencode.md
      vscode.md
      openapi.md
      codex.md
      cline.md
      continue.md
      cursor.md
      orchestra.md
```

## Generated Target Layout

```text
generated/
  agents/
    AGENTS.md
    skills/
  mcp/
    tools.json
    prompts/
    resources/
  openapi/
    openapi.yaml
  opencode/
    opencode.snippet.json
    commands/
  vscode/
    mcp.snippet.json
  codex/
    AGENTS.md
  cline/
    mcp-settings.snippet.json
  continue/
    config.snippet.yaml
  cursor/
    rules.md
  orchestra/
    pack-manifest.yaml
```

## Acceptance File Signal

Phase 3 is complete when at least three target folders are generated from the same `actlane.yaml` and tested against expected fixtures.

---

# Phase 4 — Local Policy Check

## Goal

Evaluate a tool-call JSON against generated policy bundles without a daemon or gateway.

## New Files

```text
actlane/
  packages/
    policy/
      README.md
      src-or-placeholder/
        evaluate.*
        mutate.*
        validate.*
        decision.*
        audit-record.*

    cli/
      src-or-placeholder/
        commands/
          check.*

  examples/
    04-local-policy-check/
      README.md
      actlane.yaml
      policies/
        safe-gitops-pr.policy.yaml
      generated/
        policies/
          policy-bundle.json
      tool-calls/
        allow-create-safe-draft-pr.json
        deny-secret-edit.json
        mutate-branch-prefix.json
        requires-approval.json
      expected-decisions/
        allow-result.json
        deny-result.json
        mutate-result.json
        requires-approval-result.json

  docs/
    10-local-policy-check.md
```

## Decision Output Shape

```text
examples/04-local-policy-check/expected-decisions/
  allow-result.json
  deny-result.json
  mutate-result.json
  requires-approval-result.json
```

Each decision should include:

```text
decision
reasons
mutations
originalInput
mutatedInput
auditId
policyRefs
```

## Commands Proven by Files

```bash
actlane check tool-call.json --policy generated/policies/policy-bundle.json
```

## Acceptance File Signal

Phase 4 is complete when allow, deny, mutate, and requires-approval examples are deterministic and human-readable.

---

# Phase 5 — Optional Runtime Enforcement

## Goal

Introduce runtime enforcement without turning Actlane into the primary gateway product.

## New Files

```text
actlane/
  packages/
    runtime/
      README.md
      src-or-placeholder/
        server.*
        webhook.*
        mcp-wrapper.*
        audit-sink.*
        fail-mode.*

  examples/
    05-runtime-enforcement/
      README.md
      runtime-binding.yaml
      generated/
        runtime/
          webhook-config.json
          mcp-wrapper-config.json
      requests/
        tool-call.request.json
      responses/
        allow.response.json
        deny.response.json
        mutate.response.json
      audit/
        audit.jsonl

  docs/
    runtime/
      README.md
      webhook.md
      sidecar.md
      mcp-wrapper.md
      existing-gateway-integration.md
```

## Runtime Binding Files

```text
runtime-binding.yaml
generated/runtime/webhook-config.json
generated/runtime/mcp-wrapper-config.json
```

## Acceptance File Signal

Phase 5 is complete when runtime examples prove that the same policy bundle used by `actlane check` can also be used by a webhook or MCP wrapper.

## Guardrail

Runtime files must be optional. Generation mode must not require a daemon.

---

# Phase 6 — Private Pack Distribution

## Goal

Make Actlane packs portable between developers and teams.

## New Files

```text
actlane/
  packages/
    cli/
      src-or-placeholder/
        commands/
          pack-create.*
          pack-inspect.*
          pack-install.*
          pack-remove.*
          pack-verify.*

    generator/
      src-or-placeholder/
        pack/
          archive.*
          manifest.*
          checksums.*
          lockfile.*
          verify.*

  examples/
    06-private-pack-distribution/
      README.md
      source-pack/
        actlane.yaml
        AGENTS.md
        capabilities/
        policies/
        skills/
        adapters/
        actlane.lock
        checksums.txt
      packed/
        safe-gitops-pack.zip.placeholder
      installed-project/
        .actlane/
          packs/
            safe-gitops/
          state/
            installed-packs.json

  docs/
    packs/
      README.md
      private-packs.md
      pack-lockfile.md
      checksums-and-signatures.md
```

## Pack Layout

```text
safe-gitops-pack/
  README.md
  actlane.yaml
  AGENTS.md
  capabilities/
  policies/
  skills/
  mcp/
  adapters/
  examples/
  actlane.lock
  checksums.txt
```

## Commands Proven by Files

```bash
actlane pack create packs/safe-gitops
actlane pack inspect safe-gitops-pack.zip
actlane pack install safe-gitops-pack.zip --mode overlay
actlane pack verify safe-gitops-pack.zip
actlane pack remove safe-gitops
```

## Acceptance File Signal

Phase 6 is complete when a pack can be archived, inspected, installed into a different project, verified, and safely removed.

---

# Phase 7 — Ecosystem and Enterprise Options

## Goal

Add broader ecosystem and commercial layers only after real usage proves the need.

## Possible New Files

```text
actlane/
  integrations/
    README.md
    gateways/
      generic-webhook/
        README.md
        policy-request.schema.json
        policy-response.schema.json
      kong/
        README.md
        generated-plugin-config.example.yaml
      tyk/
        README.md
        generated-policy.example.yaml
      cloudflare/
        README.md
        generated-policy.example.yaml
    registries/
      github-pack-repo.md
      internal-artifact-registry.md
      jfrog.md
    marketplaces/
      skills-marketplaces.md
      mcp-registries.md
      orchestra.md
      opencode.md

  enterprise/
    README.md
    rbac-model.md
    signed-packs.md
    audit-report-schema.md
    approval-workflow.md
    airgapped-install.md
```

## Guardrail

These files should remain proposals until users explicitly request them.

## Acceptance File Signal

Phase 7 should not start until multiple users or teams ask for private pack sharing, governance, signatures, audit reports, or gateway integration.

---

# Final Minimal Repository Before Code

If Actlane remains a no-code RFC repo, the minimal useful structure is:

```text
actlane/
  README.md
  STATUS.md
  MANIFESTO.md
  ROADMAP.md
  LICENSE

  docs/
  spec/v1alpha1/
  packs/safe-gitops/
  examples/create-safe-draft-pr/
  diagrams/
  assets/
  website/articles/

  packages/
    cli/README.md
    generator/README.md
    policy/README.md
    runtime/README.md
    sdk/README.md
```

This is enough to validate demand without pretending that a production implementation exists.
