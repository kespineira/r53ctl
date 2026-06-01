# Releasing r53ctl

Releases are automated with GitHub Actions and GoReleaser.

## One-time setup

The release workflow needs two tokens:

- `GITHUB_TOKEN`: provided automatically by GitHub Actions for publishing the GitHub Release.
- `HOMEBREW_TAP_TOKEN`: repository secret with write access to `kespineira/homebrew-tap`.

Artifact signing uses keyless cosign through the workflow's `id-token: write`
permission and Sigstore's public infrastructure, so it needs no additional
secret or key to manage.

Use a dedicated fine-grained personal access token, not the token from `gh auth token`.

Recommended token settings:

- Resource owner: `kespineira`
- Repository access: only `kespineira/homebrew-tap`
- Repository permissions: `Contents: Read and Write`
- Metadata permission: read-only, added automatically by GitHub
- Expiration: short and explicit, for example 90 days

Set or rotate the secret from the GitHub UI:

1. Create the fine-grained token at <https://github.com/settings/personal-access-tokens/new>.
2. Open <https://github.com/kespineira/r53ctl/settings/secrets/actions>.
3. Add repository secret `HOMEBREW_TAP_TOKEN`.

CLI alternative, after creating the dedicated token:

```sh
gh secret set HOMEBREW_TAP_TOKEN --repo kespineira/r53ctl
```

Paste the token when prompted. Do not pipe `gh auth token` into this command.

## Release checklist

1. Make sure `main` is clean and up to date.
2. Run local checks:

```sh
go test ./...
go vet ./...
goreleaser check
goreleaser release --snapshot --clean --skip=publish
```

3. Tag with SemVer and push the tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `Release` workflow will:

- build `r53ctl` for Linux, macOS, and Windows on `amd64` and `arm64`;
- publish `.tar.gz` archives for Linux/macOS and `.zip` archives for Windows;
- publish Linux `.deb`, `.rpm`, and `.apk` packages;
- upload `checksums.txt` and sign it with cosign (keyless via GitHub OIDC), publishing `checksums.txt.sig` and `checksums.txt.pem`;
- generate grouped release notes from commits;
- update the Homebrew cask in `kespineira/homebrew-tap`.

## Homebrew install

After a successful release:

```sh
brew install --cask kespineira/tap/r53ctl
```

## Rollback

If a release tag is wrong, delete the GitHub Release and tag, fix the issue, then push a corrected tag:

```sh
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0
```

Avoid reusing a published version once users may have installed it.
