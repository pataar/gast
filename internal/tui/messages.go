package tui

import "github.com/pataar/gast/internal/event"

// EventsFetchedMsg is sent when a background event fetch completes successfully.
type EventsFetchedMsg struct {
	Events []event.Event
}

// FetchErrorMsg is sent when a background event fetch fails.
type FetchErrorMsg struct {
	Err error
}

// CommitTitleMsg is sent when a full commit title has been resolved.
type CommitTitleMsg struct {
	EventID int    // ID of the event to update
	Title   string // full commit title
}

// TickMsg signals that it is time to start the next polling cycle.
type TickMsg struct{}
