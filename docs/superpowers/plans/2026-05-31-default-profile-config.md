# Default Profile / Configuration File — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users persist default `profile`, `region`, and `output` for `r53ctl` in a JSON config file managed via `r53ctl config`, applied automatically with predictable precedence.

**Architecture:** A new `internal/settings` package owns the JSON file (load/save/validate). The root command gains a `--config` flag and a `PersistentPreRunE` that fills unset flags from the file, respecting precedence flag > env > file > default. A `config` subcommand manages the file. A small fix in `internal/config` makes `region` honor `AWS_REGION`.

**Tech Stack:** Go 1.26, cobra, standard-library `encoding/json` (no new dependencies).

**Note for executor:** This repo has `commit.gpgsign=false` set repo-local (remote session can't reach the 1Password signing agent). Commits are unsigned by design; just `git commit -m ...`.

---

## File Structure

- `internal/settings/settings.go` (new) — `Settings` type, `Keys`, `DefaultPath`, `Load`, `Save`, `Set`, `Get`, `Unset`. Sole responsibility: the config file.
- `internal/settings/settings_test.go` (new) — unit tests for the above.
- `internal/config/config.go` (modify) — region resolution honors env.
- `internal/config/config_test.go` (new) — region precedence tests.
- `internal/cli/root.go` (modify) — `--config` flag, `app.configPath`, `settingsPath()`, `applyConfigDefaults()`, `PersistentPreRunE`, register `config` command.
- `internal/cli/config.go` (new) — `config` subcommand (`view`/`get`/`set`/`unset`/`path`).
- `internal/cli/root_test.go` (new) — `TestMain` (isolates config dir), precedence tests.
- `internal/cli/config_test.go` (new) — `config set`/`view` round-trip test.
- `README.md` (modify) — configuration section.

---

## Task 1: `internal/settings` package

**Files:**
- Create: `internal/settings/settings.go`
- Test: `internal/settings/settings_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/settings/settings_test.go`:

```go
package settings

import (
	"path/filepath"
	"testing"
)

func TestSetValidation(t *testing.T) {
	var s Settings
	if err := s.Set("output", "xml"); err == nil {
		t.Fatal("expected error for invalid output value")
	}
	if err := s.Set("bogus", "x"); err == nil {
		t.Fatal("expected error for unknown key")
	}
	if err := s.Set("profile", ""); err == nil {
		t.Fatal("expected error for empty profile value")
	}
	if err := s.Set("profile", "Domains"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Profile != "Domains" {
		t.Fatalf("profile = %q, want Domains", s.Profile)
	}
	if err := s.Set("output", "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Output != "json" {
		t.Fatalf("output = %q, want json", s.Output)
	}
}

func TestLoadMissingFile(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if (s != Settings{}) {
		t.Fatalf("expected zero value, got %#v", s)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := Settings{Profile: "Domains", Region: "eu-west-1", Output: "json"}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestGetAndUnset(t *testing.T) {
	s := Settings{Profile: "Domains"}
	v, err := s.Get("profile")
	if err != nil || v != "Domains" {
		t.Fatalf("Get(profile) = %q, %v", v, err)
	}
	if _, err := s.Get("bogus"); err == nil {
		t.Fatal("expected error for unknown key")
	}
	if err := s.Unset("profile"); err != nil {
		t.Fatalf("Unset error: %v", err)
	}
	if s.Profile != "" {
		t.Fatalf("profile after unset = %q, want empty", s.Profile)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail to compile**

Run: `go test ./internal/settings/...`
Expected: FAIL (undefined: Settings, Load, Save, etc.)

- [ ] **Step 3: Write the implementation**

Create `internal/settings/settings.go`:

```go
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Settings holds user-configurable defaults persisted to disk.
type Settings struct {
	Profile string `json:"profile,omitempty"`
	Region  string `json:"region,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Keys lists the valid configuration keys, in display order.
var Keys = []string{"profile", "region", "output"}

// DefaultPath returns the config file path, honoring XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "r53ctl", "config.json"), nil
	}
	if runtime.GOOS == "windows" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "r53ctl", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "r53ctl", "config.json"), nil
}

// Load reads settings from path. A missing file yields the zero value and no error.
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return s, nil
}

// Save writes settings to path, creating parent directories. Written with 0600 permissions.
func Save(path string, s Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// Set validates and assigns a configuration value by key.
func (s *Settings) Set(key, value string) error {
	value = strings.TrimSpace(value)
	switch key {
	case "profile":
		if value == "" {
			return fmt.Errorf("value for %q cannot be empty; use 'config unset %s' to clear", key, key)
		}
		s.Profile = value
	case "region":
		if value == "" {
			return fmt.Errorf("value for %q cannot be empty; use 'config unset %s' to clear", key, key)
		}
		s.Region = value
	case "output":
		if value != "table" && value != "json" {
			return fmt.Errorf("invalid output %q: must be table or json", value)
		}
		s.Output = value
	default:
		return fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
	return nil
}

// Get returns the stored value for key.
func (s Settings) Get(key string) (string, error) {
	switch key {
	case "profile":
		return s.Profile, nil
	case "region":
		return s.Region, nil
	case "output":
		return s.Output, nil
	default:
		return "", fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
}

// Unset clears a configuration value by key.
func (s *Settings) Unset(key string) error {
	switch key {
	case "profile":
		s.Profile = ""
	case "region":
		s.Region = ""
	case "output":
		s.Output = ""
	default:
		return fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys, ", "))
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/settings/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/settings/settings.go internal/settings/settings_test.go
git commit -m "feat: add settings package for config file"
```

---

## Task 2: region resolution honors env (`internal/config`)

**Files:**
- Modify: `internal/config/config.go:20-46`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/config/config_test.go`:

```go
package config

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadAWSHonorsRegionEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	cfg, err := LoadAWS(context.Background(), AWSOptions{})
	if err != nil {
		t.Fatalf("LoadAWS error: %v", err)
	}
	if cfg.Region != "eu-west-1" {
		t.Fatalf("region = %q, want eu-west-1", cfg.Region)
	}
}

func TestLoadAWSExplicitRegionWinsOverEnv(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	cfg, err := LoadAWS(context.Background(), AWSOptions{Region: "ap-southeast-2"})
	if err != nil {
		t.Fatalf("LoadAWS error: %v", err)
	}
	if cfg.Region != "ap-southeast-2" {
		t.Fatalf("region = %q, want ap-southeast-2", cfg.Region)
	}
}

func TestLoadAWSDefaultsRegionWhenNothingResolves(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing-creds"))
	cfg, err := LoadAWS(context.Background(), AWSOptions{})
	if err != nil {
		t.Fatalf("LoadAWS error: %v", err)
	}
	if cfg.Region != DefaultRegion {
		t.Fatalf("region = %q, want %s", cfg.Region, DefaultRegion)
	}
}
```

- [ ] **Step 2: Run tests to verify the env test fails**

Run: `go test ./internal/config/...`
Expected: `TestLoadAWSHonorsRegionEnv` FAILs (region = "us-east-1", want eu-west-1) because the current code forces `us-east-1`.

- [ ] **Step 3: Apply the fix**

In `internal/config/config.go`, replace the body of `LoadAWS` between the `loadOptions := ...` line and the `cfg, err := awsconfig.LoadDefaultConfig(...)` call. Change FROM:

```go
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	region := opts.Region
	if region == "" {
		region = DefaultRegion
	}
	loadOptions = append(loadOptions, awsconfig.WithRegion(region))
	if opts.Profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(opts.Profile))
	}
```

TO:

```go
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	if opts.Region != "" {
		loadOptions = append(loadOptions, awsconfig.WithRegion(opts.Region))
	}
	if opts.Profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(opts.Profile))
	}
```

Leave the rest unchanged; the existing post-load fallback already covers the empty case:

```go
	if cfg.Region == "" {
		cfg.Region = DefaultRegion
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "fix: honor AWS_REGION when no region flag or config is set"
```

---

## Task 3: root command wiring (`internal/cli/root.go`)

**Files:**
- Modify: `internal/cli/root.go`
- Test: `internal/cli/root_test.go` (new)

- [ ] **Step 1: Write the failing tests**

Create `internal/cli/root_test.go`:

```go
package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	r53 "github.com/kespineira/r53ctl/internal/route53"
	"github.com/kespineira/r53ctl/internal/settings"
)

// TestMain isolates the default config directory so tests never read the
// developer's real ~/.config/r53ctl/config.json.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "r53ctl-cli-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func captureFactory(captured *AWSFlags) ServiceFactory {
	return func(_ context.Context, flags AWSFlags) (r53.Service, error) {
		*captured = flags
		return &fakeService{}, nil
	}
}

func TestConfigProfileAppliedAsDefault(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := settings.Save(cfg, settings.Settings{Profile: "Domains", Region: "eu-west-1"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	var captured AWSFlags
	cmd := newRootCommand("test", io.Discard, io.Discard, captureFactory(&captured))
	cmd.SetArgs([]string{"--config", cfg, "zones", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if captured.Profile != "Domains" {
		t.Fatalf("profile = %q, want Domains", captured.Profile)
	}
	if captured.Region != "eu-west-1" {
		t.Fatalf("region = %q, want eu-west-1", captured.Region)
	}
}

func TestFlagOverridesConfigProfile(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := settings.Save(cfg, settings.Settings{Profile: "Domains"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_PROFILE", "")

	var captured AWSFlags
	cmd := newRootCommand("test", io.Discard, io.Discard, captureFactory(&captured))
	cmd.SetArgs([]string{"--config", cfg, "--profile", "prod", "zones", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if captured.Profile != "prod" {
		t.Fatalf("profile = %q, want prod", captured.Profile)
	}
}

func TestEnvOverridesConfigProfile(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := settings.Save(cfg, settings.Settings{Profile: "Domains"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_PROFILE", "envprofile")

	var captured AWSFlags
	cmd := newRootCommand("test", io.Discard, io.Discard, captureFactory(&captured))
	cmd.SetArgs([]string{"--config", cfg, "zones", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Config must NOT be applied; the SDK reads AWS_PROFILE itself, so the
	// flag value stays empty.
	if captured.Profile != "" {
		t.Fatalf("profile = %q, want empty (env should win, config not applied)", captured.Profile)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/...`
Expected: FAIL — `--config` flag unknown / config not applied (`captured.Profile` empty in the first test).

- [ ] **Step 3: Modify `internal/cli/root.go`**

Add the settings import to the import block:

```go
	appconfig "github.com/kespineira/r53ctl/internal/config"
	r53 "github.com/kespineira/r53ctl/internal/route53"
	"github.com/kespineira/r53ctl/internal/settings"
```

Add a `configPath` field to the `app` struct:

```go
type app struct {
	version        string
	awsFlags       AWSFlags
	configPath     string
	out            io.Writer
	errOut         io.Writer
	serviceFactory ServiceFactory
}
```

In `newRootCommand`, register the `--config` flag and a `PersistentPreRunE`. Add these right after the existing `cmd.PersistentFlags().StringVarP(&a.awsFlags.Output, ...)` line and before the `cmd.AddCommand(...)` calls:

```go
	cmd.PersistentFlags().StringVar(&a.configPath, "config", "", "path to r53ctl config file")

	cmd.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		return a.applyConfigDefaults(c)
	}
```

Add two methods at the end of the file:

```go
func (a *app) settingsPath() (string, error) {
	if a.configPath != "" {
		return a.configPath, nil
	}
	return settings.DefaultPath()
}

// applyConfigDefaults fills any flag the user did not set explicitly from the
// config file, unless the corresponding AWS environment variable is set.
// Precedence: flag > environment variable > config file > built-in default.
func (a *app) applyConfigDefaults(cmd *cobra.Command) error {
	path, err := a.settingsPath()
	if err != nil {
		return err
	}
	s, err := settings.Load(path)
	if err != nil {
		return err
	}
	flags := cmd.Flags()
	if s.Profile != "" && !flags.Changed("profile") && os.Getenv("AWS_PROFILE") == "" {
		a.awsFlags.Profile = s.Profile
	}
	if s.Region != "" && !flags.Changed("region") &&
		os.Getenv("AWS_REGION") == "" && os.Getenv("AWS_DEFAULT_REGION") == "" {
		a.awsFlags.Region = s.Region
	}
	if s.Output != "" && !flags.Changed("output") {
		a.awsFlags.Output = s.Output
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/...`
Expected: PASS (existing `records_test.go` tests still pass; `TestMain` isolates the config dir).

- [ ] **Step 5: Commit**

```bash
git add internal/cli/root.go internal/cli/root_test.go
git commit -m "feat: apply config file defaults with flag/env precedence"
```

---

## Task 4: `config` subcommand (`internal/cli/config.go`)

**Files:**
- Create: `internal/cli/config.go`
- Modify: `internal/cli/root.go` (register the command)
- Test: `internal/cli/config_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `internal/cli/config_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	r53 "github.com/kespineira/r53ctl/internal/route53"
)

func noopFactory(context.Context, AWSFlags) (r53.Service, error) {
	return &fakeService{}, nil
}

func TestConfigSetThenView(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")

	var out bytes.Buffer
	setCmd := newRootCommand("test", &out, io.Discard, noopFactory)
	setCmd.SetArgs([]string{"--config", cfg, "config", "set", "profile", "Domains"})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("set error: %v", err)
	}

	out.Reset()
	viewCmd := newRootCommand("test", &out, io.Discard, noopFactory)
	viewCmd.SetArgs([]string{"--config", cfg, "config", "view", "--output", "json"})
	if err := viewCmd.Execute(); err != nil {
		t.Fatalf("view error: %v", err)
	}
	if !strings.Contains(out.String(), "Domains") {
		t.Fatalf("view output = %q, want it to contain Domains", out.String())
	}
}

func TestConfigSetRejectsUnknownKey(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	cmd := newRootCommand("test", io.Discard, io.Discard, noopFactory)
	cmd.SetArgs([]string{"--config", cfg, "config", "set", "bogus", "x"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for unknown key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/...`
Expected: FAIL — unknown command `config`.

- [ ] **Step 3: Create `internal/cli/config.go`**

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kespineira/r53ctl/internal/output"
	"github.com/kespineira/r53ctl/internal/settings"
)

type configView struct {
	Path    string `json:"path"`
	Profile string `json:"profile"`
	Region  string `json:"region"`
	Output  string `json:"output"`
}

func newConfigCommand(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage r53ctl configuration defaults",
	}
	cmd.AddCommand(newConfigViewCommand(a))
	cmd.AddCommand(newConfigGetCommand(a))
	cmd.AddCommand(newConfigSetCommand(a))
	cmd.AddCommand(newConfigUnsetCommand(a))
	cmd.AddCommand(newConfigPathCommand(a))
	return cmd
}

func newConfigViewCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Show current configuration and file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if a.awsFlags.Output == "json" {
				return output.JSON(a.out, configView{
					Path:    path,
					Profile: s.Profile,
					Region:  s.Region,
					Output:  s.Output,
				})
			}
			if _, err := fmt.Fprintf(a.out, "config: %s\n", path); err != nil {
				return err
			}
			for _, key := range settings.Keys {
				value, _ := s.Get(key)
				if _, err := fmt.Fprintf(a.out, "%-8s %s\n", key, value); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newConfigGetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a single configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			value, err := s.Get(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.out, value)
			return err
		},
	}
}

func newConfigSetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if err := s.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := settings.Save(path, s); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.out, "set %s to %q\n", args[0], args[1])
			return err
		},
	}
}

func newConfigUnsetCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			s, err := settings.Load(path)
			if err != nil {
				return err
			}
			if err := s.Unset(args[0]); err != nil {
				return err
			}
			if err := settings.Save(path, s); err != nil {
				return err
			}
			_, err = fmt.Fprintf(a.out, "unset %s\n", args[0])
			return err
		},
	}
}

func newConfigPathCommand(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := a.settingsPath()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(a.out, path)
			return err
		},
	}
}
```

- [ ] **Step 4: Register the command in `internal/cli/root.go`**

After the existing `cmd.AddCommand(newRecordsCommand(a))` line, add:

```go
	cmd.AddCommand(newConfigCommand(a))
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/cli/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/config.go internal/cli/config_test.go internal/cli/root.go
git commit -m "feat: add config subcommand to manage defaults"
```

---

## Task 5: README documentation

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add a Configuration section**

In `README.md`, immediately after the `## Authentication` section (before `## Quick start`), insert:

```markdown
## Configuration

Persist default `profile`, `region`, and `output` so you don't repeat flags.
`r53ctl` stores them in `$XDG_CONFIG_HOME/r53ctl/config.json` (default
`~/.config/r53ctl/config.json`; Windows uses the OS config directory). Override
the location with `--config <path>`.

```sh
r53ctl config set profile Domains
r53ctl config set region eu-west-1
r53ctl config set output json
r53ctl config view
r53ctl config unset region
r53ctl config path
```

After setting a default profile, plain commands use it automatically:

```sh
r53ctl zones list                    # uses the saved profile
r53ctl --profile prod zones list     # the flag overrides the saved profile
```

Precedence, highest first:

1. Explicit command-line flag (`--profile`, `--region`, `--output`).
2. Environment variable (`AWS_PROFILE`, `AWS_REGION` / `AWS_DEFAULT_REGION`).
3. `config.json` value.
4. Built-in default (`output` → `table`, `region` → `us-east-1`, profile from the AWS credential chain).
```

Also add `config` to the command reference block under `## Commands`, after the
`records export` line:

```text

r53ctl config view
r53ctl config get <key>
r53ctl config set <key> <value>
r53ctl config unset <key>
r53ctl config path
```

And add `--config <path>` to the Global flags block:

```text
--config <path>        Path to r53ctl config file
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: document config command and precedence"
```

---

## Task 6: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Format check**

Run: `gofmt -l cmd internal`
Expected: no output (all files formatted). If any file is listed, run `gofmt -w cmd internal` and amend the relevant commit.

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Full test suite**

Run: `go test ./...`
Expected: all packages PASS.

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: success.

- [ ] **Step 5: Manual smoke test**

```bash
go run ./cmd/r53ctl --config /tmp/r53ctl-smoke.json config set profile Domains
go run ./cmd/r53ctl --config /tmp/r53ctl-smoke.json config view
go run ./cmd/r53ctl --config /tmp/r53ctl-smoke.json config path
rm -f /tmp/r53ctl-smoke.json
```
Expected: `set profile to "Domains"`, then a view showing `profile  Domains`, then the path.

---

## Self-review checklist (completed during authoring)

- **Spec coverage:** file location + `--config` (Task 3/4), `internal/settings` (Task 1), `config` subcommands (Task 4), precedence incl. region env fix (Task 2/3), tests (Tasks 1–4), README (Task 5). All spec sections covered.
- **Placeholders:** none — every step has complete code or exact commands.
- **Type consistency:** `Settings`, `Set`/`Get`/`Unset`/`Load`/`Save`/`DefaultPath`, `app.configPath`, `app.settingsPath()`, `app.applyConfigDefaults()`, `newConfigCommand` used consistently across tasks.
