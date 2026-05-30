# r53ctl

A small Route 53 command line tool inspired by `cli53`, focused first on hosted zones and basic record sets.

## Install from source

```sh
go install github.com/kespineira/r53ctl/cmd/r53ctl@latest
```

The project targets Go 1.24 or newer because current AWS SDK for Go v2 releases require it.

## Authentication

The CLI uses the standard AWS credential chain: environment variables, shared config, shared credentials, SSO, and instance/task roles. You can also select a profile or assume a role:

```sh
r53ctl --profile prod zones list
r53ctl --profile tooling --role-arn arn:aws:iam::123456789012:role/dns-admin zones list
```

## Commands

```sh
r53ctl zones list
r53ctl zones create example.com --comment "managed by r53ctl"
r53ctl zones delete Z1234567890 --yes

r53ctl records list example.com
r53ctl records list example.com --name www.example.com --type A
r53ctl records upsert example.com --name www.example.com --type A --ttl 300 --value 192.0.2.10
r53ctl records delete example.com --name www.example.com --type A --yes
r53ctl records export example.com --format bind
r53ctl records export example.com --format json
```

Use `--output json` for machine-readable command output:

```sh
r53ctl --output json zones list
```

## MVP scope

Supported record upserts are `A`, `AAAA`, `CAA`, `CNAME`, `MX`, `NS`, `SRV`, and `TXT`.

This first version deliberately does not implement BIND import, alias upserts, private hosted zone creation, reusable delegation sets, or Route 53 routing policies such as weighted, failover, geolocation, and latency records.

## Development

```sh
go mod tidy
go test ./...
go vet ./...
goreleaser check
```

Create a tagged release to build Linux, macOS, and Windows archives for `amd64` and `arm64`:

```sh
git tag v0.1.0
git push origin v0.1.0
```
