# CLI Adapter Sketch: create-safe-draft-pr

Status: Phase 0 placeholder. No CLI exists yet.

Future command shape:

```text
actlane run safe-gitops create-safe-draft-pr --repo <repo> --draft
```

Rules:

- require explicit confirmation before mutation;
- keep PR as draft;
- enforce `actlane/` branch prefix;
- block secret paths;
- print policy decision JSON.
