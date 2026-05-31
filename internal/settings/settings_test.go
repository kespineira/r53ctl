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
