package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FileNotExist(t *testing.T) {
	cfg, err := Load("/tmp/backctl-test-nonexistent-dir/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg.BaseURL != "" {
		t.Errorf("expected empty BaseURL, got %q", cfg.BaseURL)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `base_url: https://backstage.example.com
token_file: ~/.config/backctl/token
namespace: production
output: json
timeout: 60s
no_auth: false
verbose: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://backstage.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://backstage.example.com")
	}
	if cfg.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", cfg.Namespace, "production")
	}
	if cfg.Output != "json" {
		t.Errorf("Output = %q, want %q", cfg.Output, "json")
	}
	if cfg.Timeout != "60s" {
		t.Errorf("Timeout = %q, want %q", cfg.Timeout, "60s")
	}
	if cfg.Verbose != true {
		t.Errorf("Verbose = %v, want true", cfg.Verbose)
	}
	if cfg.NoAuth != false {
		t.Errorf("NoAuth = %v, want false", cfg.NoAuth)
	}
}

func TestLoad_TildeExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `token_file: ~/secrets/token
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "secrets/token")
	if cfg.TokenFile != expected {
		t.Errorf("TokenFile = %q, want %q", cfg.TokenFile, expected)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(":::invalid\n  - yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestDefaultPath(t *testing.T) {
	// With XDG_CONFIG_HOME set
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	got := DefaultPath()
	want := "/custom/config/backctl/config.yaml"
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}

	// Without XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", "")
	got = DefaultPath()
	home, _ := os.UserHomeDir()
	want = filepath.Join(home, ".config", "backctl", "config.yaml")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestLoad_EmptyPath_UsesDefault(t *testing.T) {
	// Just ensure it doesn't panic; file likely doesn't exist which is fine
	_, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error loading default path: %v", err)
	}
}
