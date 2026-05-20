# Actlane Generator

Status: implemented inside `packages/cli/internal/generator` for the first OpenCode MVP target.

Planned responsibility:

- load pack manifests;
- validate basic shape;
- render target artifacts;
- write deterministic output;
- create `actlane.lock`.

The current MVP generator supports only `packs/github-draft-pr-opencode` and `--target opencode`.
