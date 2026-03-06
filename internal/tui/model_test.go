package tui

import (
	"testing"

	"github.com/pataar/gast/internal/event"
)

// TestMergeEvents_DeduplicatesByID verifies that mergeEvents does not add events
// with IDs that have already been seen.
func TestMergeEvents_DeduplicatesByID(t *testing.T) {
	m := &Model{
		seenIDs: make(map[int]struct{}),
	}

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
	m := &Model{
		seenIDs: make(map[int]struct{}),
	}

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
	m := &Model{
		seenIDs: make(map[int]struct{}),
	}

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
	m := &Model{
		seenIDs: make(map[int]struct{}),
	}

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
