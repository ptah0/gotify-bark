# Git workflow

Read this guide when committing changes.

All commit messages must follow Conventional Commits:

```text
<type>[optional scope][!]: <description>
```

Use `feat` for new features, `fix` for bug fixes, and an appropriate type such as
`docs`, `refactor`, `test`, `build`, `ci`, `perf`, `style`, or `chore` for other changes.
The scope is optional. Mark breaking changes with `!` before the colon or a
`BREAKING CHANGE:` footer that explains the change.

Examples:

```text
docs: add agent workflow guidance
fix(forwarding): preserve notification titles
feat(config)!: replace legacy Bark flags with Shoutrrr URLs
```
