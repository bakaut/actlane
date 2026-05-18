# create-safe-draft-pr

Use the safe-gitops capability contract.

Rules:

- Ask for confirmation before mutating.
- Keep PR as draft.
- Use branch prefix `actlane/`.
- Do not include secret or credential files.
- Return policy decision and PR URL.
