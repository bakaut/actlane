# Policy Reference

The `create-safe-draft-pr` workflow uses:

- `safe-gitops-pr-policy`;
- `block-secrets`.

Key behavior:

- deny sensitive paths;
- require explicit confirmation for mutating actions;
- ensure branch prefix `actlane/`;
- force draft PR creation;
- audit repo, branch, files, mutations, and policy decision.
