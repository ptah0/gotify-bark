# Verification

Read this guide when validating changes.

For code changes, run from the repository root:

```sh
go test ./...
go build ./cmd/gotify-bark
```

Add a focused regression check for changed nontrivial behavior. Use the existing
`testing` and `httptest` approach in `internal/core_test.go`; tests should not
require live Gotify or Bark credentials.

For documentation-only changes, check paths, links, and claims against the
source; application tests are unnecessary. Report checks that could not run and
why.

Do not use deployment tasks as verification: `task run`, `task cleanup`, and
`task deploy` remove containers, while `task push` publishes images. See
`taskfile.yaml` before running container tasks.
