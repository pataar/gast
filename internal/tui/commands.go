package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pataar/gast/internal/gitlab"
)

// fetchEventsCmd returns a Bubble Tea command that fetches events from GitLab
// in the background. It produces either an EventsFetchedMsg or FetchErrorMsg.
func fetchEventsCmd(client *gitlab.Client, after *time.Time, pageSize int) tea.Cmd {
	return func() tea.Msg {
		events, err := client.FetchEvents(after, pageSize)
		if err != nil {
			return FetchErrorMsg{Err: err}
		}
		return EventsFetchedMsg{Events: events}
	}
}

// resolveCommitTitleCmd fetches the full commit title in the background.
func resolveCommitTitleCmd(client *gitlab.Client, eventID int, projectID int64, sha, fallback string) tea.Cmd {
	return func() tea.Msg {
		title := client.ResolveCommitTitle(projectID, sha, fallback)
		return CommitTitleMsg{EventID: eventID, Title: title}
	}
}

// tickCmd returns a Bubble Tea command that sends a TickMsg after the given
// interval, driving the periodic polling loop.
func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}
