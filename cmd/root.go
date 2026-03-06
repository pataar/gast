// Package cmd implements the CLI commands for gast using cobra.
package cmd

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pataar/gast/internal/config"
	"github.com/pataar/gast/internal/demo"
	"github.com/pataar/gast/internal/gitlab"
	"github.com/pataar/gast/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cfgFile holds the path to the configuration file provided via the --config flag.
var cfgFile string

// demoMode enables demo mode with fake data (no GitLab connection needed).
var demoMode bool

// projectFilters and groupFilters limit displayed events to matching projects/groups.
var projectFilters, groupFilters []string

// rootCmd is the top-level cobra command that launches the TUI.
var rootCmd = &cobra.Command{
	Use:   "gast",
	Short: "GitLab Activity Stream TUI",
	Long:  "A terminal UI that mirrors your GitLab dashboard activity stream with live polling.",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/gast/config.toml)")
	rootCmd.Flags().String("host", "", "GitLab host URL")
	rootCmd.Flags().String("token", "", "GitLab personal access token")
	rootCmd.Flags().Duration("interval", 0, "poll interval (e.g. 30s)")
	rootCmd.Flags().Bool("full-project-path", false, "show full project path instead of short name")
	rootCmd.Flags().StringSliceVar(&projectFilters, "project", nil, "filter to projects matching these names (comma-separated)")
	rootCmd.Flags().StringSliceVar(&groupFilters, "group", nil, "filter to groups matching these prefixes (comma-separated)")
	rootCmd.Flags().BoolVar(&demoMode, "demo", false, "run with fake data (no GitLab connection)")

	viper.BindPFlag("gitlab_host", rootCmd.Flags().Lookup("host"))
	viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))
	viper.BindPFlag("poll_interval", rootCmd.Flags().Lookup("interval"))
	viper.BindPFlag("show_full_project_path", rootCmd.Flags().Lookup("full-project-path"))
}

// run loads configuration, initializes the GitLab client, and starts the Bubble Tea program.
func run(cmd *cobra.Command, args []string) error {
	if demoMode {
		return runDemo()
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	client, err := gitlab.NewClient(cfg.GitLabHost, cfg.Token)
	if err != nil {
		return fmt.Errorf("gitlab client error: %w", err)
	}

	if username, err := client.CurrentUsername(); err == nil {
		cfg.Username = username
	}

	model := tui.NewModel(client, cfg)
	model.SetFilters(projectFilters, groupFilters)
	return runTUI(model)
}

// runDemo starts the TUI with fake data and no GitLab connection.
func runDemo() error {
	cfg := &config.Config{
		PollInterval: 30 * time.Second,
		PageSize:     50,
		Username:     "pieter.willekens",
	}

	model := tui.NewDemoModel(cfg, demo.Events())
	return runTUI(model)
}

// runTUI starts a Bubble Tea program with the given model.
func runTUI(model tea.Model) error {
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// Execute runs the root command and exits with code 1 on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
