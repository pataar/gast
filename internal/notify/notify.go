// Package notify provides cross-platform desktop notifications.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Send sends a desktop notification with the given title and body.
// If url is non-empty, clicking the notification opens that URL (requires
// terminal-notifier on macOS).
func Send(title, body, url string) error {
	switch runtime.GOOS {
	case "darwin":
		return sendDarwin(title, body, url)
	case "linux":
		return sendLinux(title, body, url)
	default:
		return nil
	}
}

// sendDarwin uses terminal-notifier when available (supports click-to-open and
// persistent display), otherwise falls back to osascript.
func sendDarwin(title, body, url string) error {
	if tn, err := exec.LookPath("terminal-notifier"); err == nil {
		args := []string{
			"-title", title,
			"-message", body,
			"-timeout", "0", // persistent until dismissed
		}
		if url != "" {
			args = append(args, "-open", url)
		}
		return exec.Command(tn, args...).Start()
	}
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	return exec.Command("osascript", "-e", script).Start()
}

// sendLinux uses notify-send with critical urgency so the notification persists.
func sendLinux(title, body, url string) error {
	args := []string{"-u", "critical", title, body}
	return exec.Command("notify-send", args...).Start()
}

// FormatMention creates a notification body from author, project, and snippet.
func FormatMention(author, project, snippet string) string {
	var b strings.Builder
	b.WriteString(author)
	b.WriteString(" mentioned you in ")
	b.WriteString(project)
	if snippet != "" {
		b.WriteString(": ")
		if len(snippet) > 100 {
			snippet = snippet[:97] + "..."
		}
		b.WriteString(snippet)
	}
	return b.String()
}
