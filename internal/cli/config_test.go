package cli

import (
	"bytes"
	"context"
	"io"
	"os"
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

func TestConfigGetUnsetPathRoundTrip(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	run := func(args ...string) (string, error) {
		var out bytes.Buffer
		cmd := newRootCommand("test", &out, io.Discard, noopFactory)
		cmd.SetArgs(append([]string{"--config", cfg}, args...))
		err := cmd.Execute()
		return out.String(), err
	}

	if _, err := run("config", "set", "profile", "Domains"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := run("config", "get", "profile")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(got) != "Domains" {
		t.Fatalf("get profile = %q, want Domains", strings.TrimSpace(got))
	}
	if _, err := run("config", "unset", "profile"); err != nil {
		t.Fatalf("unset: %v", err)
	}
	got, err = run("config", "get", "profile")
	if err != nil {
		t.Fatalf("get after unset: %v", err)
	}
	if strings.TrimSpace(got) != "" {
		t.Fatalf("get profile after unset = %q, want empty", strings.TrimSpace(got))
	}
	gotPath, err := run("config", "path")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if strings.TrimSpace(gotPath) != cfg {
		t.Fatalf("path = %q, want %q", strings.TrimSpace(gotPath), cfg)
	}
}

func TestConfigViewRejectsInvalidOutput(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{"output":"xml"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand("test", io.Discard, io.Discard, noopFactory)
	cmd.SetArgs([]string{"--config", cfg, "config", "view"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for invalid output value in config")
	}
}

func TestConfigViewReportsMalformedFile(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{invalid`), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCommand("test", io.Discard, io.Discard, noopFactory)
	cmd.SetArgs([]string{"--config", cfg, "config", "view"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for malformed config file")
	}
	if !strings.Contains(err.Error(), cfg) {
		t.Fatalf("error %q should mention the config path %q", err.Error(), cfg)
	}
}
