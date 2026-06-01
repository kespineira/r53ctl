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
