# GitHub Draft PR Instructions

Capability: create-github-draft-pr

Before mutating repository or GitHub state, check the Actlane policy bundle and ask for explicit confirmation.

Defaults:

- draft: true
- branch prefix: actlane/
- maximum files: 20
- maximum diff bytes: 120000

Return a concise summary with policy decision, changed files, final branch, and draft PR URL.
