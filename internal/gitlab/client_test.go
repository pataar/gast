package gitlab

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// newTestClient returns a Client backed by a fake GitLab API that serves project lookups and counts API calls.
func newTestClient(t *testing.T) (*Client, *atomic.Int64) {
	t.Helper()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		id := strings.TrimPrefix(r.URL.Path, "/api/v4/projects/")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id": %s, "path_with_namespace": "group/project-%s"}`, id, id)
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, "test-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client, &calls
}

// TestResolveProjects_ResolvesAllIDs verifies that every requested project ID gets a resolved name.
func TestResolveProjects_ResolvesAllIDs(t *testing.T) {
	client, calls := newTestClient(t)

	ids := map[int64]struct{}{1: {}, 2: {}, 3: {}}
	names := client.resolveProjects(ids)

	if len(names) != len(ids) {
		t.Fatalf("got %d names, want %d", len(names), len(ids))
	}
	for id := range ids {
		want := fmt.Sprintf("group/project-%d", id)
		if names[id] != want {
			t.Errorf("names[%d] = %q, want %q", id, names[id], want)
		}
	}
	if got := calls.Load(); got != int64(len(ids)) {
		t.Errorf("API calls = %d, want %d", got, len(ids))
	}
}

// TestResolveProjects_UsesCache verifies that a second resolve for the same IDs makes no further API calls.
func TestResolveProjects_UsesCache(t *testing.T) {
	client, calls := newTestClient(t)

	ids := map[int64]struct{}{7: {}, 8: {}}
	client.resolveProjects(ids)
	callsAfterFirst := calls.Load()

	names := client.resolveProjects(ids)

	if got := calls.Load(); got != callsAfterFirst {
		t.Errorf("second resolve made %d extra API calls, want 0", got-callsAfterFirst)
	}
	if names[7] != "group/project-7" {
		t.Errorf("names[7] = %q, want %q", names[7], "group/project-7")
	}
}
