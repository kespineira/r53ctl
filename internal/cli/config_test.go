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
