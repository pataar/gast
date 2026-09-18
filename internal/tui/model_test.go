package tui

import (
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/pataar/gast/internal/config"
	"github.com/pataar/gast/internal/event"
)

// newTestModel builds a minimal Model with the state mergeEvents and friends require.
func newTestModel() *Model {
	return &Model{seenIDs: make(map[int]struct{})}
}

// TestMergeEvents_DeduplicatesByID verifies that mergeEvents does not add events with already-seen IDs.
func TestMergeEvents_DeduplicatesByID(t *testing.T) {
	m := newTestModel()

	initial := []event.Event{
		{ID: 1, AuthorUsername: "alice"},
		{ID: 2, AuthorUsername: "bob"},
	}
	m.mergeEvents(initial)

	if len(m.events) != 2 {
		t.Fatalf("after initial merge: got %d events, want 2", len(m.events))
	}

	// Merge again with a duplicate and a new event.
	m.mergeEvents([]event.Event{
		{ID: 2, AuthorUsername: "bob-duplicate"},
		{ID: 3, AuthorUsername: "carol"},
	})

	if len(m.events) != 3 {
		t.Fatalf("after second merge: got %d events, want 3", len(m.events))
	}

	// Verify the duplicate was not added (bob's name should be original).
	for _, e := range m.events {
		if e.ID == 2 && e.AuthorUsername != "bob" {
			t.Errorf("duplicate ID 2 was overwritten: got author %q, want %q", e.AuthorUsername, "bob")
		}
	}
}

// TestMergeEvents_AppendsNewEvents verifies that new events are appended so
// the newest events appear last in the slice (chronological order).
func TestMergeEvents_AppendsNewEvents(t *testing.T) {
	m := newTestModel()

	m.mergeEvents([]event.Event{
		{ID: 1, AuthorUsername: "first"},
	})

	m.mergeEvents([]event.Event{
		{ID: 2, AuthorUsername: "second"},
	})

	if len(m.events) != 2 {
		t.Fatalf("got %d events, want 2", len(m.events))
	}

	// Oldest event (ID 1) at index 0, newest (ID 2) at the end.
	if m.events[0].ID != 1 {
		t.Errorf("events[0].ID = %d, want 1 (oldest first)", m.events[0].ID)
	}
	if m.events[1].ID != 2 {
		t.Errorf("events[1].ID = %d, want 2 (newest last)", m.events[1].ID)
	}
}

// TestMergeEvents_CapsAtMaxEvents verifies that the events slice never exceeds
// the maxEvents limit (500).
func TestMergeEvents_CapsAtMaxEvents(t *testing.T) {
	m := newTestModel()

	// Fill up to maxEvents. API returns newest-first, so we build the batch
	// with descending IDs to mimic real API order.
	batch := make([]event.Event, maxEvents)
	for i := range batch {
		batch[i] = event.Event{ID: maxEvents - i} // 500, 499, ..., 1
	}
	m.mergeEvents(batch)

	if len(m.events) != maxEvents {
		t.Fatalf("got %d events, want %d", len(m.events), maxEvents)
	}

	// Add 10 more newer events (API order: newest first).
	extra := make([]event.Event, 10)
	for i := range extra {
		extra[i] = event.Event{ID: maxEvents + 10 - i} // 510, 509, ..., 501
	}
	m.mergeEvents(extra)

	if len(m.events) != maxEvents {
		t.Fatalf("after overflow: got %d events, want %d", len(m.events), maxEvents)
	}

	// The oldest 10 events (IDs 1-10) should have been trimmed from the front.
	if m.events[0].ID != 11 {
		t.Errorf("events[0].ID = %d, want 11 (oldest trimmed)", m.events[0].ID)
	}

	// The newest event should be at the end.
	if m.events[len(m.events)-1].ID != 510 {
		t.Errorf("events[last].ID = %d, want 510 (newest last)", m.events[len(m.events)-1].ID)
	}
}

// TestMergeEvents_CleansUpSeenIDs verifies that seenIDs entries are removed
// for events that get trimmed when the list exceeds maxEvents.
func TestMergeEvents_CleansUpSeenIDs(t *testing.T) {
	m := newTestModel()

	// Fill up to maxEvents (API order: newest first → descending IDs).
	batch := make([]event.Event, maxEvents)
	for i := range batch {
		batch[i] = event.Event{ID: maxEvents - i} // 500, 499, ..., 1
	}
	m.mergeEvents(batch)

	// Add 10 more newer events to trigger trimming.
	extra := make([]event.Event, 10)
	for i := range extra {
		extra[i] = event.Event{ID: maxEvents + 10 - i} // 510, 509, ..., 501
	}
	m.mergeEvents(extra)

	// The oldest 10 events (IDs 1-10) should have been trimmed from the
	// front of the slice and removed from seenIDs.
	for i := 1; i <= 10; i++ {
		if _, exists := m.seenIDs[i]; exists {
			t.Errorf("seenIDs still contains removed event ID %d", i)
		}
	}

	// A newer event should still be tracked.
	if _, exists := m.seenIDs[maxEvents+5]; !exists {
		t.Errorf("seenIDs missing event ID %d that should still be present", maxEvents+5)
	}

	// seenIDs count should match events count.
	if len(m.seenIDs) != len(m.events) {
		t.Errorf("seenIDs length = %d, events length = %d; they should match", len(m.seenIDs), len(m.events))
	}
}

// TestShouldSuppressNotifications verifies that notifications are suppressed
// during the initial fetch and the first fetch after clearing events.
func TestShouldSuppressNotifications(t *testing.T) {
	t.Run("initial fetch", func(t *testing.T) {
		m := newTestModel()
		m.initialFetch = true
		if !m.shouldSuppressNotifications() {
			t.Error("expected suppression during initial fetch")
		}
	})

	t.Run("first fetch after clear", func(t *testing.T) {
		now := time.Now()
		m := newTestModel()
		m.clearedAt = &now
		if !m.shouldSuppressNotifications() {
			t.Error("expected suppression on first fetch after clear")
		}
	})

	t.Run("subsequent fetch after clear", func(t *testing.T) {
		now := time.Now()
		m := newTestModel()
		m.clearedAt = &now
		m.events = []event.Event{{ID: 1}}
		if m.shouldSuppressNotifications() {
			t.Error("should not suppress when events already present after clear")
		}
	})

	t.Run("normal fetch", func(t *testing.T) {
		m := newTestModel()
		if m.shouldSuppressNotifications() {
			t.Error("should not suppress during normal fetch")
		}
	})
}

// botTestEvents holds a human event, a plain bot event, and a bot event mentioning "pieter".
func botTestEvents() []event.Event {
	return []event.Event{
		{ID: 1, AuthorUsername: "alice"},
		{ID: 2, AuthorUsername: "renovate-bot", NoteBody: "updated deps"},
		{ID: 3, AuthorUsername: "renovate-bot", NoteBody: "ping @pieter"},
	}
}

// pressKey sends a single printable key press through Update and returns the resulting model.
func pressKey(t *testing.T, m Model, pressed rune) Model {
	t.Helper()
	updated, _ := m.Update(tea.KeyPressMsg{Code: pressed, Text: string(pressed)})
	return updated.(Model)
}

func TestBuildDisplayItems_HidesBotsUnlessTheyMentionUser(t *testing.T) {
	event.CurrentUser = "pieter"
	t.Cleanup(func() { event.CurrentUser = "" })

	m := newTestModel()
	m.hideBots = true
	// mergeEvents receives events newest-first.
	m.mergeEvents(botTestEvents())
	m.buildDisplayItems()

	if len(m.events) != 3 {
		t.Fatalf("got %d stored events, want 3 (hidden bot events stay in memory)", len(m.events))
	}
	if len(m.displayItems) != 2 {
		t.Fatalf("filter on: got %d display items, want 2 (human + mentioning bot)", len(m.displayItems))
	}
	for _, item := range m.displayItems {
		if item.primaryEvent.ID == 2 {
			t.Error("filter on: non-mentioning bot event is displayed")
		}
	}
	if m.hiddenBotCount != 1 {
		t.Errorf("hiddenBotCount = %d, want 1", m.hiddenBotCount)
	}

	m.hideBots = false
	m.buildDisplayItems()
	if len(m.displayItems) != 3 {
		t.Fatalf("filter off: got %d display items, want 3", len(m.displayItems))
	}
	if m.hiddenBotCount != 0 {
		t.Errorf("filter off: hiddenBotCount = %d, want 0", m.hiddenBotCount)
	}
}

func TestToggleBotsKey_TogglesFilterAndKeepsSelection(t *testing.T) {
	event.CurrentUser = "pieter"
	t.Cleanup(func() { event.CurrentUser = "" })

	// NewModel sets event.CurrentUser from the config, so the username must be passed here.
	m := NewDemoModel(&config.Config{FilterBots: true, Username: "pieter"}, nil)
	if !m.hideBots {
		t.Fatal("hideBots should start from the filter_bots config value")
	}
	m.mergeEvents(botTestEvents())
	m.buildDisplayItems()
	// Chronological order is 3, 2, 1; with bots hidden the items are [3, 1]. Select event 1.
	m.selectedIdx = 1

	m = pressKey(t, m, 'b')
	if m.hideBots {
		t.Fatal("pressing b should show bot events")
	}
	if len(m.displayItems) != 3 {
		t.Fatalf("got %d display items after toggle, want 3", len(m.displayItems))
	}
	if selected, _ := m.selectedEvent(); selected.ID != 1 {
		t.Errorf("selection moved to event %d, want it to stay on event 1", selected.ID)
	}

	// Select the plain bot event, then hide bots again: selection must stay in range.
	m.selectedIdx = 1
	m = pressKey(t, m, 'b')
	if !m.hideBots {
		t.Fatal("pressing b again should hide bot events")
	}
	if _, ok := m.selectedEvent(); !ok {
		t.Errorf("selectedIdx %d is out of range after hiding the selected event", m.selectedIdx)
	}
}

func TestClearKey_ResetsHiddenBotCount(t *testing.T) {
	t.Cleanup(func() { event.CurrentUser = "" })

	m := NewDemoModel(&config.Config{FilterBots: true, Username: "pieter"}, nil)
	m.mergeEvents(botTestEvents())
	m.buildDisplayItems()
	if m.hiddenBotCount != 1 {
		t.Fatalf("setup: hiddenBotCount = %d, want 1", m.hiddenBotCount)
	}

	m = pressKey(t, m, 'c')
	if m.hiddenBotCount != 0 {
		t.Errorf("hiddenBotCount = %d after clear, want 0", m.hiddenBotCount)
	}
}

// mentionTestModel returns a demo model holding, oldest to newest: a mention (1), a plain event (2),
// the user's own comment quoting their handle (3), a plain event (4), and another mention (5).
func mentionTestModel(t *testing.T) Model {
	t.Helper()
	t.Cleanup(func() { event.CurrentUser = "" })

	m := NewDemoModel(&config.Config{Username: "pieter"}, nil)
	m.mergeEvents([]event.Event{
		{ID: 5, AuthorUsername: "bob", NoteBody: "@pieter can you review?"},
		{ID: 4, AuthorUsername: "alice"},
		{ID: 3, AuthorUsername: "pieter", NoteBody: "cc @pieter"},
		{ID: 2, AuthorUsername: "alice", NoteBody: "looks good"},
		{ID: 1, AuthorUsername: "alice", NoteBody: "ping @pieter"},
	})
	m.buildDisplayItems()
	return m
}

func displayedIDs(m Model) []int {
	ids := make([]int, len(m.displayItems))
	for i, item := range m.displayItems {
		ids[i] = item.primaryEvent.ID
	}
	return ids
}

func TestMentionsOnlyKey_ShowsOnlyMentionsByOthers(t *testing.T) {
	m := mentionTestModel(t)

	m = pressKey(t, m, 'm')
	if got := displayedIDs(m); !slices.Equal(got, []int{1, 5}) {
		t.Fatalf("mentions only: displayed IDs = %v, want [1 5]", got)
	}

	m = pressKey(t, m, 'm')
	if got := displayedIDs(m); len(got) != 5 {
		t.Fatalf("mentions view off: displayed IDs = %v, want all 5", got)
	}
}

func TestNextMentionKey_JumpsForwardAndWraps(t *testing.T) {
	m := mentionTestModel(t)
	m.selectedIdx = 1

	m = pressKey(t, m, 'n')
	if selected, _ := m.selectedEvent(); selected.ID != 5 {
		t.Fatalf("first n: selected event %d, want 5", selected.ID)
	}

	m = pressKey(t, m, 'n')
	if selected, _ := m.selectedEvent(); selected.ID != 1 {
		t.Fatalf("second n: selected event %d, want 1 (wrap around)", selected.ID)
	}
}

func TestNextMentionKey_WithoutMentionsKeepsSelection(t *testing.T) {
	m := NewDemoModel(&config.Config{}, nil)
	m.mergeEvents([]event.Event{{ID: 2, AuthorUsername: "alice"}, {ID: 1, AuthorUsername: "bob"}})
	m.buildDisplayItems()
	m.selectedIdx = 1

	m = pressKey(t, m, 'n')
	if m.selectedIdx != 1 {
		t.Errorf("selectedIdx = %d, want 1", m.selectedIdx)
	}
}

func TestEventsFetched_KeepsSelectionWhenOldEventsAreTrimmed(t *testing.T) {
	sized, _ := NewDemoModel(&config.Config{}, nil).Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m := sized.(Model)
	existing := make([]event.Event, maxEvents)
	for i := range existing {
		// mergeEvents receives events newest-first.
		existing[i] = event.Event{ID: maxEvents - i, AuthorUsername: "alice"}
	}
	m.mergeEvents(existing)
	m.buildDisplayItems()
	m.selectedIdx = 299
	if selected, _ := m.selectedEvent(); selected.ID != 300 {
		t.Fatalf("setup: selected event %d, want 300", selected.ID)
	}

	newEvents := make([]event.Event, 10)
	for i := range newEvents {
		newEvents[i] = event.Event{ID: maxEvents + 10 - i, AuthorUsername: "bob"}
	}
	updated, _ := m.Update(EventsFetchedMsg{Events: newEvents})
	m = updated.(Model)

	if selected, _ := m.selectedEvent(); selected.ID != 300 {
		t.Errorf("selected event %d after the fetch trimmed old events, want 300", selected.ID)
	}
}
