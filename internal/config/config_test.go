package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestLoad_MissingGitLabHost verifies that loading a config without gitlab_host returns an error.
func TestLoad_MissingGitLabHost(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte(`token = "glpat-abc"`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for missing gitlab_host, got nil")
	}
	if got := err.Error(); got != "gitlab_host is required — run 'gast configure' or set GITLAB_ACTIVITY_HOST" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

// TestLoad_MissingToken verifies that loading a config without token returns an error.
func TestLoad_MissingToken(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte(`gitlab_host = "https://gitlab.example.com"`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
	if got := err.Error(); got != "token is required — run 'gast configure' or set GITLAB_ACTIVITY_TOKEN" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

// TestLoad_InvalidPollInterval verifies that a non-parseable poll_interval returns an error.
func TestLoad_InvalidPollInterval(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	content := `gitlab_host = "https://gitlab.example.com"
token = "glpat-abc"
poll_interval = "not-a-duration"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for invalid poll_interval, got nil")
	}
}

// TestLoad_ValidConfig verifies that a complete, valid TOML config loads correctly.
func TestLoad_ValidConfig(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	content := `gitlab_host = "https://gitlab.example.com"
token = "glpat-abc123"
poll_interval = "15s"
page_size = 25
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitLabHost != "https://gitlab.example.com" {
		t.Errorf("GitLabHost = %q, want %q", cfg.GitLabHost, "https://gitlab.example.com")
	}
	if cfg.Token != "glpat-abc123" {
		t.Errorf("Token = %q, want %q", cfg.Token, "glpat-abc123")
	}
	if cfg.PollInterval != 15*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 15*time.Second)
	}
	if cfg.PageSize != 25 {
		t.Errorf("PageSize = %d, want %d", cfg.PageSize, 25)
	}
}

// TestLoad_EnvVarsOverrideConfigFile verifies that environment variables take precedence
// over values defined in the config file.
func TestLoad_EnvVarsOverrideConfigFile(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	content := `gitlab_host = "https://gitlab.example.com"
token = "glpat-file-token"
poll_interval = "15s"
page_size = 25
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GITLAB_ACTIVITY_HOST", "https://override.example.com")
	t.Setenv("GITLAB_ACTIVITY_TOKEN", "glpat-env-token")
	t.Setenv("GITLAB_ACTIVITY_INTERVAL", "45s")
	t.Setenv("GITLAB_ACTIVITY_PAGE_SIZE", "100")

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GitLabHost != "https://override.example.com" {
		t.Errorf("GitLabHost = %q, want %q", cfg.GitLabHost, "https://override.example.com")
	}
	if cfg.Token != "glpat-env-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "glpat-env-token")
	}
	if cfg.PollInterval != 45*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 45*time.Second)
	}
	if cfg.PageSize != 100 {
		t.Errorf("PageSize = %d, want %d", cfg.PageSize, 100)
	}
}

// TestLoad_DefaultValues verifies that poll_interval defaults to 30s and page_size defaults to 50.
func TestLoad_DefaultValues(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	content := `gitlab_host = "https://gitlab.example.com"
token = "glpat-abc"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PollInterval != 30*time.Second {
		t.Errorf("PollInterval = %v, want default %v", cfg.PollInterval, 30*time.Second)
	}
	if cfg.PageSize != 50 {
		t.Errorf("PageSize = %d, want default %d", cfg.PageSize, 50)
	}
}

// TestLoad_MinimumPollInterval verifies that a short but valid poll interval like "5s" is accepted.
func TestLoad_MinimumPollInterval(t *testing.T) {
	viper.Reset()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	content := `gitlab_host = "https://gitlab.example.com"
token = "glpat-abc"
poll_interval = "5s"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PollInterval != 5*time.Second {
		t.Errorf("PollInterval = %v, want %v", cfg.PollInterval, 5*time.Second)
	}
}
