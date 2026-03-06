// Package notify provides cross-platform desktop notifications.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Send sends a desktop notification with the given title and body.
func Send(title, body string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		return exec.Command("osascript", "-e", script).Start()
	case "linux":
		return exec.Command("notify-send", title, body).Start()
	default:
		return nil
	}
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
