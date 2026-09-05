package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/pataar/gast/internal/config"
	"github.com/pataar/gast/internal/gitlab"
	"github.com/pataar/gast/internal/notify"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

/*
configureCmd represents the configure subcommand that launches an interactive
configuration wizard for setting up GitLab connection details and preferences.
*/
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Interactive configuration wizard for gast",
	Long: `Launch an interactive wizard that guides you through setting up your
GitLab connection. The resulting configuration is written to the XDG config
directory (~/.config/gast/config.toml).`,
	RunE: runConfigure,
}

func init() {
	// Register configure as a subcommand of the root command.
	rootCmd.AddCommand(configureCmd)
}

// ── Styles – kept minimal; just enough colour to guide the user.

var (
	styleHeading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8C00"))
	styleSuccess = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00"))
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	styleError   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	stylePrompt  = lipgloss.NewStyle().Bold(true)
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

// ── Configuration wizard entry-point

/*
runConfigure is the main handler for the `gast configure` command.
It walks the user through each config field, validates the token against the
GitLab API, and writes the final config to disk.
*/
func runConfigure(cmd *cobra.Command, args []string) error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println(styleHeading.Render("gast configuration wizard"))
	fmt.Println(styleDim.Render("Press Enter to accept the default value shown in [brackets].\n"))

	// Step 0: Check for an existing configuration and load defaults from it.
	configDir := config.Dir()
	configPath := config.FilePath()

	existing := loadExistingConfig(configPath)
	if existing != nil {
		fmt.Println(styleWarning.Render("An existing configuration was found at " + configPath))
		if !promptYesNo(scanner, "Do you want to overwrite it?", true) {
			fmt.Println("Aborted.")
			return nil
		}
		fmt.Println()
	} else {
		existing = &config.Config{}
	}
	if existing.PollInterval <= 0 {
		existing.PollInterval = 30 * time.Second
	}
	if existing.PageSize <= 0 {
		existing.PageSize = 50
	}

	host, err := promptHost(scanner, existing.GitLabHost)
	if err != nil {
		return err
	}

	token, err := promptToken(scanner, existing.Token)
	if err != nil {
		return err
	}

	interval, err := promptInterval(scanner, existing.PollInterval.String())
	if err != nil {
		return err
	}

	pageSize, err := promptPageSize(scanner, existing.PageSize)
	if err != nil {
		return err
	}

	showFullProject := promptYesNo(scanner, "Show full project path (e.g. org/group/project)?", existing.ShowFullProject)

	notifications := promptYesNo(scanner, "Enable desktop notifications for @mentions?", existing.Notifications)

	if notifications && !notify.CheckDarwinDeps() {
		fmt.Println(styleWarning.Render("  terminal-notifier is required for macOS notifications."))
		if promptYesNo(scanner, "  Install it now with Homebrew?", true) {
			fmt.Print("  Installing terminal-notifier... ")
			if err := installTerminalNotifier(); err != nil {
				fmt.Println(styleError.Render("FAILED: " + err.Error()))
				fmt.Println(styleWarning.Render("  You can install it manually: brew install terminal-notifier"))
			} else {
				fmt.Println(styleSuccess.Render("OK"))
			}
		} else {
			fmt.Println(styleWarning.Render("  Notifications won't work until you run: brew install terminal-notifier"))
		}
	}

	// Validate the token by calling the GitLab API.
	fmt.Print("\nValidating token against " + styleDim.Render(host) + " ... ")
	username, err := validateToken(host, token)
	if err != nil {
		fmt.Println(styleError.Render("FAILED"))
		return fmt.Errorf("token validation failed: %w", err)
	}
	fmt.Println(styleSuccess.Render("OK") + " (authenticated as " + stylePrompt.Render(username) + ")")

	// Write the configuration file.
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	content := buildTOML(host, token, interval, pageSize, showFullProject, notifications)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Println(styleSuccess.Render("\nConfiguration saved to " + configPath))
	return nil
}

// ── Prompt helpers

// promptHost asks the user for a GitLab host URL and validates that it includes an http(s) scheme.
func promptHost(scanner *bufio.Scanner, defaultVal string) (string, error) {
	for {
		input := prompt(scanner, "GitLab host URL", defaultVal)
		if input == "" {
			fmt.Println(styleError.Render("  Host URL is required."))
			continue
		}

		parsed, err := url.Parse(input)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			fmt.Println(styleError.Render("  Invalid URL — must include scheme (e.g. https://gitlab.example.com)."))
			continue
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			fmt.Println(styleError.Render("  Scheme must be http or https."))
			continue
		}

		// Normalise: strip trailing slash.
		return strings.TrimRight(input, "/"), nil
	}
}

/*
promptToken asks the user for a personal access token. An existing token is shown masked in the
label (never as the prompt default, so it can't echo back). It warns when the conventional
glpat- / glpat_ prefix is missing.
*/
func promptToken(scanner *bufio.Scanner, defaultVal string) (string, error) {
	label := "Personal access token"
	if defaultVal != "" {
		label += " [" + defaultVal[:6] + strings.Repeat("*", len(defaultVal)-6) + "]"
	}

	for {
		input := prompt(scanner, label, "")
		if input == "" {
			input = defaultVal
		}

		if input == "" {
			fmt.Println(styleError.Render("  Token is required."))
			continue
		}

		if !strings.HasPrefix(input, "glpat-") && !strings.HasPrefix(input, "glpat_") {
			fmt.Println(styleWarning.Render("  Warning: token does not start with glpat- or glpat_. Continuing anyway."))
		}

		return input, nil
	}
}

// promptInterval asks for a poll interval and validates it as a Go duration of at least 5 seconds.
func promptInterval(scanner *bufio.Scanner, defaultVal string) (string, error) {
	for {
		input := prompt(scanner, "Poll interval", defaultVal)

		dur, err := time.ParseDuration(input)
		if err != nil {
			fmt.Println(styleError.Render("  Invalid duration — use Go duration syntax (e.g. 30s, 1m)."))
			continue
		}
		if dur < 5*time.Second {
			fmt.Println(styleError.Render("  Minimum poll interval is 5s."))
			continue
		}

		return input, nil
	}
}

// promptPageSize asks for the number of events to fetch per poll, between 1 and 100.
func promptPageSize(scanner *bufio.Scanner, defaultVal int) (int, error) {
	for {
		input := prompt(scanner, "Page size (1-100)", strconv.Itoa(defaultVal))

		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > 100 {
			fmt.Println(styleError.Render("  Must be a number between 1 and 100."))
			continue
		}

		return n, nil
	}
}

// promptYesNo asks a yes/no question and returns the boolean answer.
func promptYesNo(scanner *bufio.Scanner, question string, defaultYes bool) bool {
	hint := "Y/n"
	if !defaultYes {
		hint = "y/N"
	}

	// The hint doubles as the prompt default; on empty input it falls through to defaultYes below.
	input := strings.ToLower(prompt(scanner, question, hint))

	switch input {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultYes
	}
}

/*
prompt prints a styled prompt line (with the default value in [brackets] when present) and reads
one line of input from the scanner. If the user enters nothing, the default value is returned.
*/
func prompt(scanner *bufio.Scanner, label, defaultVal string) string {
	suffix := ": "
	if defaultVal != "" {
		suffix = " [" + defaultVal + "]: "
	}
	fmt.Print(stylePrompt.Render(label) + suffix)

	if !scanner.Scan() {
		return defaultVal
	}

	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return defaultVal
	}
	return text
}

// ── Dependency installation

// installTerminalNotifier runs `brew install terminal-notifier`.
func installTerminalNotifier() error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("Homebrew not found")
	}
	cmd := exec.Command(brew, "install", "terminal-notifier")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ── Token validation via the GitLab REST API

// validateToken checks the token by requesting the authenticated user and returns their username.
func validateToken(host, token string) (string, error) {
	client, err := gitlab.NewClient(host, token)
	if err != nil {
		return "", err
	}
	return client.CurrentUsername()
}

// ── Config file I/O

/*
loadExistingConfig reads an existing config file into a Config used to pre-populate wizard
defaults. Returns nil if the file does not exist or cannot be parsed.
*/
func loadExistingConfig(path string) *config.Config {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil
	}
	cfg := &config.Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil
	}
	return cfg
}

// buildTOML constructs the TOML content for the configuration file.
func buildTOML(host, token, interval string, pageSize int, showFullProject, notifications bool) string {
	var b strings.Builder
	b.WriteString("# gast configuration — generated by `gast configure`\n\n")
	fmt.Fprintf(&b, "gitlab_host = %q\n", host)
	fmt.Fprintf(&b, "notifications = %t\n", notifications)
	fmt.Fprintf(&b, "page_size = %d\n", pageSize)
	fmt.Fprintf(&b, "poll_interval = %q\n", interval)
	fmt.Fprintf(&b, "show_full_project_path = %t\n", showFullProject)
	fmt.Fprintf(&b, "token = %q\n", token)
	return b.String()
}
