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

// TickMsg signals that it is time to start the next polling cycle.
type TickMsg struct{}
