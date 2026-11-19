# Contributing to log2json

## Prerequisites

- Go 1.21 or later
- [golangci-lint](https://golangci-lint.run/welcome/install/) (for linting)

## Development Workflow

1. Fork and clone the repository
2. Create a feature branch: `git checkout -b my-feature`
3. Make your changes
4. Run all checks: `make check`
5. Commit and push
6. Open a pull request against `main`

### Layered review pilot

For an enrolled change, open a draft using the [pilot PR template](.github/PULL_REQUEST_TEMPLATE/layered-review.md).
Copy it into the PR description and link evidence from the tested revision.
Run the two [review passes](REVIEW.md) before requesting human review. Check
the new head after a push and mark stale evidence. No pilot tier grants merge
authority. Existing repository rules decide who can approve.

Record the proposed tier and whether automation alone would have cleared that
commit **before** a human responds. Record material corrections humans catch
on that revision. A change to parsed types or output is ordinary at minimum.
Release automation, CI permissions and irreversible effects need owner review.

Compare consecutive pilot PRs with a comparable baseline by tier. Track median
and p95 time waiting for human review, active author/reviewer minutes, review
rounds and unique findings fixed, dismissed or open. Link released commits to
unplanned reverts, incidents, severity, pages and responder minutes. Report raw
counts and rates per 100 merged PRs over an equal observation window. Mark
unreleased changes pending. Extend the pilot if data is sparse. Any proposal
to reduce approvals needs a separate maintainer decision after a rollback drill
and evidence of time saved without worse release or on-call outcomes.

## Make Targets

| Target         | Description                              |
|---------------|------------------------------------------|
| `make build`    | Build the binary to `build/`           |
| `make test`     | Run tests with race detection          |
| `make lint`     | Run golangci-lint                      |
| `make vet`      | Run go vet                             |
| `make check`    | Run lint + vet + test (pre-push gate)  |
| `make coverage` | Generate coverage report               |
| `make version`  | Show current version (from git tags)   |
| `make help`     | List all targets                       |

## Code Standards

- All code must pass `make check` before merge
- Tests use stdlib `testing` only (no test frameworks)
- Zero external dependencies -- stdlib only
- Table-driven tests with `t.Run` subtests
- Follow existing code patterns in `internal/`

## Adding a New Log Format Parser

1. Create `internal/parser/yourformat_parser.go` implementing the `Parser` interface
2. Create `internal/parser/yourformat_parser_test.go` with table-driven tests
3. Register the parser in `internal/parser/registry.go` inside `NewRegistry()`
4. Add a sample file in `testdata/sample_yourformat.log`
5. Run `make check` to validate

The `Parser` interface (defined in `internal/parser/parser.go`):

```go
type Parser interface {
    Name() string
    Description() string
    CanParse(line string) bool
    Parse(line string) (*Entry, error)
}
```

## CI Pipeline

Every push and pull request triggers:

- **Test**: Build and test on 3 OS (Linux, macOS, Windows) x 3 Go versions (1.21, 1.22, 1.23)
- **Lint**: golangci-lint with 12 linters
- **Cross-compile**: Builds for linux/darwin/windows x amd64/arm64

## Releasing

Releases are automated via GoReleaser. A maintainer runs:

```bash
make release V=x.y.z
git push origin vx.y.z
```

This creates an annotated git tag and pushes it, which triggers the release workflow to build cross-platform binaries and publish a GitHub Release.
