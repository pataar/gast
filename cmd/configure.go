package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/pataar/gast/internal/config"
	"github.com/spf13/cobra"
)

// configureCmd represents the configure subcommand that launches an interactive
// configuration wizard for setting up GitLab connection details and preferences.
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

// --------------------------------------------------------------------------
// Styles – kept minimal; just enough colour to guide the user.
// --------------------------------------------------------------------------

var (
	styleHeading = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF8C00"))
	styleSuccess = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00CC00"))
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
	styleError   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000"))
	stylePrompt  = lipgloss.NewStyle().Bold(true)
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

// --------------------------------------------------------------------------
// Configuration wizard entry-point
// --------------------------------------------------------------------------

// runConfigure is the main handler for the `gast configure` command.
// It walks the user through each config field, validates the token against the
// GitLab API, and writes the final config to disk.
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
	}

	// Step 1: GitLab host URL
	hostDefault := ""
	if existing != nil {
		hostDefault = existing.host
	}
	host, err := promptHost(scanner, hostDefault)
	if err != nil {
		return err
	}

	// Step 2: Personal access token
	tokenDefault := ""
	if existing != nil {
		tokenDefault = existing.token
	}
	token, err := promptToken(scanner, tokenDefault)
	if err != nil {
		return err
	}

	// Step 3: Poll interval
	intervalDefault := "30s"
	if existing != nil && existing.pollInterval != "" {
		intervalDefault = existing.pollInterval
	}
	interval, err := promptInterval(scanner, intervalDefault)
	if err != nil {
		return err
	}

	// Step 4: Page size
	pageSizeDefault := 50
	if existing != nil && existing.pageSize > 0 {
		pageSizeDefault = existing.pageSize
	}
	pageSize, err := promptPageSize(scanner, pageSizeDefault)
	if err != nil {
		return err
	}

	// Step 5: Show full project path
	showFullProjectDefault := false
	if existing != nil {
		showFullProjectDefault = existing.showFullProject
	}
	showFullProject := promptYesNo(scanner, "Show full project path (e.g. org/group/project)?", showFullProjectDefault)

	// Step 6: Desktop notifications
	notificationsDefault := false
	if existing != nil {
		notificationsDefault = existing.notifications
	}
	notifications := promptYesNo(scanner, "Enable desktop notifications for @mentions?", notificationsDefault)

	// Step 7: Validate the token by calling the GitLab API.
	fmt.Print("\nValidating token against " + styleDim.Render(host) + " ... ")
	username, err := validateToken(host, token)
	if err != nil {
		fmt.Println(styleError.Render("FAILED"))
		return fmt.Errorf("token validation failed: %w", err)
	}
	fmt.Println(styleSuccess.Render("OK") + " (authenticated as " + stylePrompt.Render(username) + ")")

	// Step 8: Write the configuration file.
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

// --------------------------------------------------------------------------
// Prompt helpers
// --------------------------------------------------------------------------

// promptHost asks the user for a GitLab host URL and validates that it
// includes a scheme (http or https).
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

// promptToken asks the user for a personal access token. It validates the
// conventional glpat- / glpat_ prefix and warns if the prefix is missing.
func promptToken(scanner *bufio.Scanner, defaultVal string) (string, error) {
	displayDefault := defaultVal
	if displayDefault != "" {
		// Mask the token in the prompt so it is not leaked on screen.
		displayDefault = displayDefault[:6] + strings.Repeat("*", len(displayDefault)-6)
	}

	for {
		input := prompt(scanner, "Personal access token", displayDefault)

		// If the user pressed Enter without typing, reuse the stored default.
		if input == displayDefault && defaultVal != "" {
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

// promptInterval asks the user for a poll interval string and validates it
// as a Go duration with a minimum of 5 seconds.
func promptInterval(scanner *bufio.Scanner, defaultVal string) (string, error) {
	for {
		input := prompt(scanner, "Poll interval", defaultVal)
		if input == "" {
			input = defaultVal
		}

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

// promptPageSize asks the user for the number of events to fetch per poll
// and validates the value is between 1 and 100.
func promptPageSize(scanner *bufio.Scanner, defaultVal int) (int, error) {
	for {
		input := prompt(scanner, "Page size (1-100)", strconv.Itoa(defaultVal))
		if input == "" {
			return defaultVal, nil
		}

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

	input := prompt(scanner, question+" ["+hint+"]", "")
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultYes
	}
}

// prompt prints a styled prompt line and reads one line of input from the
// scanner. If the user enters nothing, the default value is returned.
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

// --------------------------------------------------------------------------
// Token validation via the GitLab REST API
// --------------------------------------------------------------------------

// validateToken performs a GET request to /api/v4/user using the provided host
// and token. It returns the authenticated username on success.
func validateToken(host, token string) (string, error) {
	reqURL := host + "/api/v4/user"

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	// We only need the username from the response.
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if body.Username == "" {
		return "", fmt.Errorf("response did not contain a username")
	}

	return body.Username, nil
}

// --------------------------------------------------------------------------
// Config file I/O
// --------------------------------------------------------------------------

// existingConfig holds values parsed from a previously saved configuration
// file, used to pre-populate defaults in the wizard.
type existingConfig struct {
	host            string
	notifications   bool
	pageSize        int
	pollInterval    string
	showFullProject bool
	token           string
}

// loadExistingConfig attempts to read an existing config.toml and extract the
// known keys. Returns nil if the file does not exist or cannot be parsed.
func loadExistingConfig(path string) *existingConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	cfg := &existingConfig{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")

		switch key {
		case "gitlab_host":
			cfg.host = val
		case "notifications":
			cfg.notifications = val == "true"
		case "page_size":
			cfg.pageSize, _ = strconv.Atoi(val)
		case "poll_interval":
			cfg.pollInterval = val
		case "show_full_project_path":
			cfg.showFullProject = val == "true"
		case "token":
			cfg.token = val
		}
	}

	return cfg
}

// buildTOML constructs the TOML content for the configuration file.
func buildTOML(host, token, interval string, pageSize int, showFullProject, notifications bool) string {
	var b strings.Builder
	b.WriteString("# gast configuration — generated by `gast configure`\n\n")
	b.WriteString(fmt.Sprintf("gitlab_host = %q\n", host))
	b.WriteString(fmt.Sprintf("notifications = %t\n", notifications))
	b.WriteString(fmt.Sprintf("page_size = %d\n", pageSize))
	b.WriteString(fmt.Sprintf("poll_interval = %q\n", interval))
	b.WriteString(fmt.Sprintf("show_full_project_path = %t\n", showFullProject))
	b.WriteString(fmt.Sprintf("token = %q\n", token))
	return b.String()
}
