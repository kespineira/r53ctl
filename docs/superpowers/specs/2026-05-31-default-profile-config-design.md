# Design: default profile / configuration file

Date: 2026-05-31
Status: Approved (pending implementation plan)

## Problem

`r53ctl` reads `--profile`, `--region`, and `--output` on every invocation.
A user who always works against the same AWS profile (e.g. `Domains`) must
either repeat `--profile Domains` on every command or `export AWS_PROFILE`
in their shell. They want `r53ctl` to own a persistent default it applies
automatically, while still letting flags and environment variables override
it for one-off commands.

## Goals

- Persist user defaults for `profile`, `region`, and `output` in a file
  `r53ctl` manages itself.
- Manage that file through `r53ctl config` subcommands (no hand-editing
  required, though hand-editing is allowed).
- Keep precedence predictable and aligned with AWS conventions.

## Non-goals (YAGNI)

- `role-arn` / `endpoint-url` in the config file (they vary per command).
- Multiple named contexts / profiles-of-defaults.
- Config import/export.
- New runtime dependencies (use the standard library only).

## Mechanism

Config file managed by `r53ctl`, in JSON (no new dependency; `encoding/json`
from the standard library).

### File location

- Default: `$XDG_CONFIG_HOME/r53ctl/config.json`; if `XDG_CONFIG_HOME` is
  unset, `~/.config/r53ctl/config.json`.
- Windows: fall back to `os.UserConfigDir()` joined with `r53ctl/config.json`.
- Override: a new global flag `--config <path>`. This also makes the feature
  testable without touching the real home directory.

### File semantics

- Missing file is treated as an empty config (no error).
- Invalid JSON returns a clear error naming the path.
- Unknown keys in the file are ignored (forward compatibility).
- Written with `0600` permissions; parent directory created with `MkdirAll`.

## Components

### New package `internal/settings`

Single responsibility: read/write/validate the JSON settings file. The
existing `internal/config` package keeps its single responsibility (AWS SDK
config loading) and is not modified by this design.

```go
package settings

type Settings struct {
    Profile string `json:"profile,omitempty"`
    Region  string `json:"region,omitempty"`
    Output  string `json:"output,omitempty"`
}

// Keys lists the valid configuration keys, in display order.
var Keys = []string{"profile", "region", "output"}

func DefaultPath() (string, error)          // XDG / ~/.config / Windows fallback
func Load(path string) (Settings, error)    // missing file -> zero value, nil error
func Save(path string, s Settings) error    // MkdirAll + write 0600

func (s *Settings) Set(key, value string) error // validates key and value
func (s Settings) Get(key string) (string, error)
```

Validation in `Set`:
- Unknown key -> error listing the valid keys.
- `output` accepts only `table` or `json`.
- `profile` and `region` accept any non-empty string.

### New command `internal/cli/config.go`

```text
r53ctl config view             # show current settings + resolved file path
r53ctl config get <key>        # print a single value
r53ctl config set <key> <val>  # load, set, save, confirm
r53ctl config unset <key>      # clear a single key
r53ctl config path             # print the resolved file path
```

`view` respects the global `--output` (table or json). `set`/`unset` load the
current file (or empty), mutate, and save.

### Region resolution fix (`internal/config/config.go`)

For the precedence table below to hold, `region` must honor `AWS_REGION` /
`AWS_DEFAULT_REGION` when neither a flag nor the config file provides one.
Today `LoadAWS` unconditionally calls `WithRegion`, forcing `us-east-1` and
shadowing the environment. Change it to:

- Pass `WithRegion(opts.Region)` only when `opts.Region != ""`.
- Keep the existing post-load fallback `if cfg.Region == "" { cfg.Region = DefaultRegion }`,
  which now actually runs and applies `us-east-1` only when the SDK resolved
  nothing.

This is a small, contained change required by this feature; it does not alter
behavior when `--region` or a config region is set.

### Root command changes (`internal/cli/root.go`)

- Add a persistent flag `--config <path>` (default empty).
- Add a `PersistentPreRunE` that:
  1. Resolves the config path (`--config` value, else `settings.DefaultPath()`).
  2. Loads settings (missing file -> empty).
  3. For each of `profile`, `region`, `output`: if the corresponding flag was
     NOT explicitly set (`cmd.Flags().Changed(name)` is false) AND the relevant
     AWS environment variable is NOT set, apply the config value to
     `a.awsFlags`.
- Register `newConfigCommand(a)` alongside the existing `zones` and `records`
  commands.

## Precedence

Per setting, highest wins:

1. Explicit command-line flag (`cmd.Flags().Changed(name)`).
2. Environment variable:
   - `profile`: `AWS_PROFILE`
   - `region`: `AWS_REGION`, then `AWS_DEFAULT_REGION`
   - `output`: none (no AWS env var)
3. `config.json` value.
4. Built-in default: `output` -> `table`; `region` -> `us-east-1`
   (via existing `config.LoadAWS`); `profile` -> AWS SDK default chain.

Rationale: an explicit `export AWS_PROFILE=...` should win over a saved
default, matching AWS tooling conventions and avoiding the footgun of a
stored profile silently overriding an environment variable the user just set.

## Error handling

- Invalid JSON in the file: error naming the path; surfaced on any command
  that loads it (including `config view`).
- `config set` with an unknown key or invalid `output` value: error before
  writing anything.
- Save failures (permissions, disk): error wrapping the path.
- An invalid `output` value that somehow reaches command execution is still
  caught by the existing check in `app.service`.

## Testing

- `internal/settings/settings_test.go`:
  - `Set` validation: valid keys, unknown key, invalid `output` value.
  - `Load` of a missing file returns zero value and no error.
  - `Load`/`Save` round-trip.
  - `Get` returns stored values.
- `internal/cli/config_test.go`:
  - `config set` followed by `config view` using `--config` pointed at a temp
    file.
  - Precedence, reusing the injectable `serviceFactory` already in the root
    command: config value is applied; an explicit `--profile` flag overrides
    it; `AWS_PROFILE` (via `t.Setenv`) overrides the config value. Same shape
    for `region` with `AWS_REGION`.
- `internal/config/config_test.go`:
  - With no `--region` and no config, `AWS_REGION` (via `t.Setenv`) is honored
    and `us-east-1` is used only when nothing resolves.

These tests also raise the currently-thin coverage of the CLI and config
layers.

## Documentation

Add a "Configuration / default profile" section to `README.md` documenting the
`config` subcommands, the file location, and the precedence table.

## Files

- `internal/settings/settings.go` (new)
- `internal/settings/settings_test.go` (new)
- `internal/cli/config.go` (new)
- `internal/cli/config_test.go` (new)
- `internal/cli/root.go` (modify: `--config` flag, `PersistentPreRunE`,
  register `config` command)
- `internal/config/config.go` (modify: region resolution honors env)
- `internal/config/config_test.go` (new: region env precedence)
- `README.md` (modify: configuration section)
