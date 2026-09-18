// Package event defines the domain types and formatting logic for GitLab
// activity events displayed in the TUI.
package event

import (
	"regexp"
	"strings"
	"time"
)

// ponytail: username heuristic because the events API exposes no bot flag; look up the Users API `bot` field if this misfires.
var botUsernamePattern = regexp.MustCompile(`(?i)^((project|group)_\d+_bot|service_account_)|([-_]bot|\[bot\])$`)

// IsBotUsername reports whether the username looks like a bot: GitLab access-token/service accounts or a bot suffix.
func IsBotUsername(username string) bool {
	return botUsernamePattern.MatchString(username)
}

// PushData holds details about a git push event, including the number of commits and the branch or tag reference.
type PushData struct {
	CommitCount int
	CommitTo    string // SHA of the head commit
	Ref         string
	CommitTitle string
}

// IsIssueTargetType returns true when the target type represents an issue (Issue, or the newer WorkItem).
func IsIssueTargetType(targetType string) bool {
	switch strings.ToLower(targetType) {
	case "issue", "workitem":
		return true
	}
	return false
}

// IsNoteTargetType returns true when the target type represents a comment (Note, DiscussionNote, or DiffNote).
func IsNoteTargetType(targetType string) bool {
	switch strings.ToLower(targetType) {
	case "note", "discussionnote", "diffnote":
		return true
	}
	return false
}

// Event represents a single GitLab user contribution event, normalized from
// the GitLab API response into a display-friendly structure.
type Event struct {
	ID             int
	ActionName     string
	AuthorUsername string
	CreatedAt      time.Time
	NoteBody       string // Snippet of the comment body (for "commented on" events).
	NoteID         int    // ID of the note itself (used for #note_<id> URL anchors).
	NoteableType   string // Parent type for notes: "Issue", "MergeRequest", etc.
	NoteableIID    int    // Parent IID for notes (may differ from TargetIID).
	ProjectID      int64
	ProjectName    string
	PushData       *PushData
	TargetIID      int
	TargetTitle    string
	TargetType     string
}
