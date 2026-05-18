# Security Policy

Actlane is currently pre-alpha and documentation-first. Do not use it as a production security control.

## Supported Versions

No production versions are supported yet.

## Reporting Security Issues

If you find a security issue in repository content, examples, policies, or future implementation code, open a private security advisory if available for the repository. If private reporting is not available, open an issue with minimal sensitive detail and ask maintainers to move discussion to a private channel.

## Current Security Posture

Actlane currently provides:

- design documents;
- diagrams;
- early specification notes;
- example policy concepts.

Actlane does not currently provide:

- runtime enforcement;
- production access control;
- production audit guarantees;
- signed pack verification;
- hosted policy decision service.

## Intended Security Model

Actlane aims to make safe agent actions more reviewable and reproducible by describing:

- allowed actions;
- denied paths and inputs;
- safe default mutations;
- approval requirements;
- generated artifacts;
- owned files and blocks;
- lockfile checksums;
- audit metadata.

Those capabilities are design goals until implemented and tested.
