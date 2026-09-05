// Package notify provides cross-platform desktop notifications.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/pataar/gast/internal/event"
)

var (
	darwinWarned bool
	darwinMu     sync.Mutex

	notifierOnce sync.Once
	notifierPath string
	notifierErr  error
)

/*
Send sends a desktop notification with the given title and body.
On macOS, requires terminal-notifier (brew install terminal-notifier).
If url is non-empty, clicking the notification opens that URL.
*/
func Send(title, body, url string) error {
	switch runtime.GOOS {
	case "darwin":
		return sendDarwin(title, body, url)
	case "linux":
		return sendLinux(title, body)
	default:
		return nil
	}
}

// lookupNotifier resolves the terminal-notifier binary once and caches the result for the process lifetime.
func lookupNotifier() (string, error) {
	notifierOnce.Do(func() {
		notifierPath, notifierErr = exec.LookPath("terminal-notifier")
	})
	return notifierPath, notifierErr
}

// CheckDarwinDeps returns true if terminal-notifier is available on macOS; on other platforms always true.
func CheckDarwinDeps() bool {
	if runtime.GOOS != "darwin" {
		return true
	}
	_, err := lookupNotifier()
	return err == nil
}

/*
sendDarwin uses terminal-notifier for notifications. If terminal-notifier
is not installed, it prints a one-time warning and skips the notification.
*/
func sendDarwin(title, body, url string) error {
	tn, err := lookupNotifier()
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
	return startAndReap(exec.Command(tn, args...))
}

// sendLinux uses notify-send with critical urgency so the notification persists.
func sendLinux(title, body string) error {
	return startAndReap(exec.Command("notify-send", "-u", "critical", title, body))
}

// startAndReap starts cmd without blocking and waits for it in the background so no zombie process is left.
func startAndReap(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// FormatMention creates a notification body from author, project, and snippet.
func FormatMention(author, project, snippet string) string {
	msg := fmt.Sprintf("%s mentioned you in %s", author, project)
	if snippet != "" {
		msg += ": " + event.Truncate(snippet, 100)
	}
	return msg
}
