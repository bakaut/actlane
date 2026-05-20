# github-draft-pr

Use after changes are reviewed and the user asks to prepare a GitHub draft PR.

## Safety Contract

- Create only GitHub draft pull requests.
- Require explicit user confirmation before mutation.
- Force branch prefix actlane/.
- Deny secret-like paths: **/*credential*, **/*secret*, **/*token*, **/id_ed25519, **/id_rsa, .env, .env.*.
- Never include files outside the reviewed file list.
- Prefer running configured project tests before creating the PR.
