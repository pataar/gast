// Package browser provides cross-platform browser opening.
package browser

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pataar/gast/internal/event"
)

// OpenEvent opens the GitLab URL for the given event in the default browser.
func OpenEvent(host string, e event.Event) error {
	u := EventURL(host, e)
	if u == "" {
		return nil
	}
	return Open(u)
}

// EventURL constructs the GitLab web URL for an event based on its type.
func EventURL(host string, e event.Event) string {
	host = strings.TrimRight(host, "/")
	project := e.ProjectName
	if project == "" {
		return ""
	}

	switch {
	case e.PushData != nil:
		if e.PushData.CommitTo != "" {
			return fmt.Sprintf("%s/%s/-/commit/%s", host, project, e.PushData.CommitTo)
		}
		return fmt.Sprintf("%s/%s/-/tree/%s", host, project, e.PushData.Ref)
	case strings.EqualFold(e.TargetType, "MergeRequest") && e.TargetIID > 0:
		return fmt.Sprintf("%s/%s/-/merge_requests/%d", host, project, e.TargetIID)
	case strings.EqualFold(e.TargetType, "Issue") && e.TargetIID > 0:
		return fmt.Sprintf("%s/%s/-/issues/%d", host, project, e.TargetIID)
	case strings.EqualFold(e.TargetType, "WorkItem") && e.TargetIID > 0:
		return fmt.Sprintf("%s/%s/-/issues/%d", host, project, e.TargetIID)
	case strings.EqualFold(e.TargetType, "Note") || strings.EqualFold(e.TargetType, "DiscussionNote"):
		return noteURL(host, project, e)
	default:
		return fmt.Sprintf("%s/%s", host, project)
	}
}

// noteURL constructs the URL for a comment event using the note's parent type.
func noteURL(host, project string, e event.Event) string {
	iid := e.NoteableIID
	if iid == 0 {
		iid = e.TargetIID
	}
	nt := strings.ToLower(e.NoteableType)
	switch {
	case (nt == "issue" || nt == "workitem") && iid > 0:
		return fmt.Sprintf("%s/%s/-/issues/%d", host, project, iid)
	case nt == "mergerequest" && iid > 0:
		return fmt.Sprintf("%s/%s/-/merge_requests/%d", host, project, iid)
	case iid > 0:
		// Fallback when NoteableType is empty (older API responses).
		return fmt.Sprintf("%s/%s/-/merge_requests/%d", host, project, iid)
	default:
		return fmt.Sprintf("%s/%s", host, project)
	}
}

// Open opens the given URL in the default browser.
func Open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
