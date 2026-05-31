# Contributing to r53ctl

Thanks for considering a contribution.

## Development setup

Requirements:

- Go 1.26.3 or newer
- GoReleaser 2.x for release config validation

Run the standard checks:

```sh
go mod tidy
gofmt -w cmd internal
go test ./...
go vet ./...
goreleaser check
```

Validate release packaging locally without publishing:

```sh
HOMEBREW_TAP_TOKEN=dummy goreleaser release --snapshot --clean --skip=publish
```

## Pull requests

- Keep changes focused and small where possible.
- Add or update tests for behavior changes.
- Keep destructive Route 53 operations explicit; commands that delete resources should require confirmation flags.
- Do not add live AWS calls to the default test suite.
- Update README or docs when changing user-facing commands, flags, or release behavior.

## Commit style

Conventional commit prefixes are encouraged because release notes are grouped from commit messages:

- `feat:`
- `fix:`
- `docs:`
- `ci:`
- `chore:`
- `refactor:`
- `test:`

## Releases

Maintainers publish releases by pushing SemVer tags:

```sh
git tag v0.1.0
git push origin v0.1.0
```

See [docs/releasing.md](docs/releasing.md).
