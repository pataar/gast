// Package config handles loading and validating application configuration
// from files, environment variables, and CLI flags via viper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds the runtime configuration for connecting to GitLab and
// controlling polling behavior.
type Config struct {
	GitLabHost   string        `mapstructure:"gitlab_host"`
	Token        string        `mapstructure:"token"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	PageSize     int           `mapstructure:"page_size"`
	Username     string        // Resolved at startup from the API; not persisted.
}

// Dir returns the default configuration directory path for gast.
// Uses os.UserConfigDir (XDG_CONFIG_HOME on Linux, ~/Library/Application Support
// on macOS, %AppData% on Windows), falling back to ~/.config.
func Dir() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "gast")
}

// FilePath returns the full path to the default config file.
func FilePath() string {
	return filepath.Join(Dir(), "config.toml")
}

// Load reads configuration from file, environment variables, and defaults.
// If cfgFile is non-empty, it is used as the config file path; otherwise
// the default XDG location is used.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(Dir())
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetDefault("poll_interval", "30s")
	viper.SetDefault("page_size", 50)

	viper.SetEnvPrefix("GITLAB_ACTIVITY")
	viper.BindEnv("gitlab_host", "GITLAB_ACTIVITY_HOST")
	viper.BindEnv("token", "GITLAB_ACTIVITY_TOKEN")
	viper.BindEnv("poll_interval", "GITLAB_ACTIVITY_INTERVAL")
	viper.BindEnv("page_size", "GITLAB_ACTIVITY_PAGE_SIZE")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	cfg := &Config{}

	cfg.GitLabHost = viper.GetString("gitlab_host")
	cfg.Token = viper.GetString("token")
	cfg.PageSize = viper.GetInt("page_size")

	interval := viper.GetString("poll_interval")
	dur, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid poll_interval %q: %w", interval, err)
	}
	cfg.PollInterval = dur

	if cfg.GitLabHost == "" {
		return nil, fmt.Errorf("gitlab_host is required — run 'gast configure' or set GITLAB_ACTIVITY_HOST")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("token is required — run 'gast configure' or set GITLAB_ACTIVITY_TOKEN")
	}

	return cfg, nil
}
