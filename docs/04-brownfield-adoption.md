# Brownfield Adoption

Actlane must be safe to try in an existing project.

The adoption lifecycle is:

```text
inspect -> init -> generate -> diff -> plan -> apply -> remove
```

## Default Behavior

By default, Actlane should:

```text
inspect existing files
create .actlane/ only
generate into .actlane/generated
show diffs before apply
avoid overwriting files
write only owned blocks
record ownership metadata
remove only owned files and blocks
```

## Ownership Markers

If Actlane writes into an existing file, the content must be inside owned markers:

```markdown
<!-- actlane:start create-safe-draft-pr -->
Generated content
<!-- actlane:end create-safe-draft-pr -->
```

## State Files

Future adoption state may live in:

```text
.actlane/state/
.actlane/backups/
.actlane/reports/
actlane.lock
```

## Remove Must Be Trustworthy

Removal should never delete arbitrary project content.

It may remove:

- files listed as Actlane-owned in the lockfile;
- blocks marked with Actlane ownership markers;
- generated files under `.actlane/generated`.

It should refuse or warn when a generated file has been manually changed.
