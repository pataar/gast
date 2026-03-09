// Package notify provides cross-platform desktop notifications.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

var (
	darwinWarned bool
	darwinMu     sync.Mutex
)

// Send sends a desktop notification with the given title and body.
// On macOS, requires terminal-notifier (brew install terminal-notifier).
// If url is non-empty, clicking the notification opens that URL.
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

// CheckDarwinDeps returns true if terminal-notifier is available on macOS.
// On other platforms it always returns true.
func CheckDarwinDeps() bool {
	if runtime.GOOS != "darwin" {
		return true
	}
	_, err := exec.LookPath("terminal-notifier")
	return err == nil
}

// sendDarwin uses terminal-notifier for notifications. If terminal-notifier
// is not installed, it prints a one-time warning and skips the notification.
func sendDarwin(title, body, url string) error {
	tn, err := exec.LookPath("terminal-notifier")
	if err != nil {
		darwinMu.Lock()
		defer darwinMu.Unlock()
		if !darwinWarned {
			darwinWarned = true
			fmt.Println("Warning: terminal-notifier not found — notifications disabled. Install with: brew install terminal-notifier")
		}
		return nil
	}
	args := []string{
		"-title", title,
		"-message", body,
		"-timeout", "0",
		"-sound", "default",
	}
	if url != "" {
		args = append(args, "-open", url)
	}
	return exec.Command(tn, args...).Start()
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
