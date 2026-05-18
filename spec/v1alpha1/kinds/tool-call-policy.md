# ToolCallPolicy

`ToolCallPolicy` describes safety behavior for a capability.

It answers:

```text
what is allowed
what is denied
what defaults are mutated
what requires approval
what must be audited
```

## Common Rules

- repository allowlist;
- forbidden paths;
- branch prefix;
- maximum files changed;
- maximum diff size;
- draft PR default;
- explicit confirmation for mutating actions.

## Decisions

Future local policy checks should return:

```text
allow
deny
mutate
requires-approval
```
