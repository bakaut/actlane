# Private Packs

Actlane starts with portable private capability packs, not a public marketplace.

## Why Private First

Teams often need to share internal agent capabilities:

```text
safe GitHub workflows
CI troubleshooting
release notes
incident response
internal docs lookup
repository maintenance
```

These workflows usually contain private conventions:

- repository allowlists;
- branch prefixes;
- forbidden paths;
- internal tool names;
- approval rules;
- audit requirements;
- target runtime preferences.

## Pack Portability

A private pack should be inspectable before use.

It should include:

```text
README.md
actlane.yaml
capabilities/
policies/
skills/
mcp/
adapters/
examples/
actlane.lock
checksums.txt
```

## First Pack

The first pack is:

```text
safe-gitops
```

The first workflow is:

```text
create-safe-draft-pr
```

This is concrete enough to validate whether developers want portable safe agent workflows before building a registry, marketplace, or SaaS.
