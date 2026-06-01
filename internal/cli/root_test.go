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
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func captureFactory(captured *AWSFlags) ServiceFactory {
	return func(_ context.Context, flags AWSFlags) (r53.Service, error) {
		*captured = flags
		return &fakeService{}, nil
	}
}

// TestExecuteContextReachesServiceFactory guards the contract that the command
// execution context flows to the service factory, so a cancelled context
// (e.g. from a SIGINT-bound context in main) propagates to AWS calls.
func TestExecuteContextReachesServiceFactory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var gotErr error
	factory := func(c context.Context, _ AWSFlags) (r53.Service, error) {
		gotErr = c.Err()
		return &fakeService{}, nil
	}
	cmd := newRootCommand("test", io.Discard, io.Discard, factory)
	cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "config.json"), "zones", "list"})
	_ = cmd.ExecuteContext(ctx)

	if gotErr == nil {
		t.Fatal("service factory did not receive the execution context; cancellation will not propagate")
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

func TestConfigOutputAppliedAsDefault(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := settings.Save(cfg, settings.Settings{Output: "json"}); err != nil {
		t.Fatal(err)
	}

	var captured AWSFlags
	cmd := newRootCommand("test", io.Discard, io.Discard, captureFactory(&captured))
	cmd.SetArgs([]string{"--config", cfg, "zones", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if captured.Output != "json" {
		t.Fatalf("output = %q, want json", captured.Output)
	}
}

func TestEnvOverridesConfigRegion(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := settings.Save(cfg, settings.Settings{Region: "eu-west-1"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_REGION", "us-east-2")
	t.Setenv("AWS_DEFAULT_REGION", "")

	var captured AWSFlags
	cmd := newRootCommand("test", io.Discard, io.Discard, captureFactory(&captured))
	cmd.SetArgs([]string{"--config", cfg, "zones", "list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Config region must NOT be applied; the SDK reads AWS_REGION itself.
	if captured.Region != "" {
		t.Fatalf("region = %q, want empty (env should win, config not applied)", captured.Region)
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
