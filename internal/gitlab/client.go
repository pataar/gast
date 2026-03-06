// Package gitlab provides a client for fetching activity events from a
// GitLab instance via its REST API.
package gitlab

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pataar/gast/internal/event"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the go-gitlab API client and maintains a cache of project ID
// to path-with-namespace mappings to reduce API calls.
type Client struct {
	api          *gl.Client
	projectCache map[int64]string
	mu           sync.RWMutex
}

// NewClient creates a new GitLab API client configured with the given host URL
// and personal access token.
func NewClient(host, token string) (*Client, error) {
	api, err := gl.NewClient(token, gl.WithBaseURL(host+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("creating gitlab client: %w", err)
	}
	return &Client{
		api:          api,
		projectCache: make(map[int64]string),
	}, nil
}

// FetchEvents retrieves activity events across all projects the authenticated
// user is a member of (mirroring /dashboard/activity?filter=projects).
// When after is non-nil, only events created after that time are returned.
// Results are sorted in descending order and limited to pageSize entries.
func (c *Client) FetchEvents(after *time.Time, pageSize int) ([]event.Event, error) {
	opts := &gl.ListContributionEventsOptions{
		ListOptions: gl.ListOptions{Page: 1, PerPage: int64(pageSize)},
		Sort:        gl.Ptr("desc"),
		Scope:       gl.Ptr("all"),
	}
	if after != nil {
		t := gl.ISOTime(*after)
		opts.After = &t
	}

	raw, _, err := c.api.Events.ListCurrentUserContributionEvents(opts)
	if err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}

	events := make([]event.Event, 0, len(raw))
	for _, re := range raw {
		// Skip push events that are merge commits or have no actual commits.
		if re.PushData.CommitCount > 0 && isMergeCommit(re.PushData.CommitTitle) {
			continue
		}
		if re.PushData.CommitCount == 0 && re.PushData.Ref != "" {
			continue
		}

		e := event.Event{
			ID:             int(re.ID),
			ActionName:     re.ActionName,
			AuthorUsername: re.AuthorUsername,
			TargetIID:      int(re.TargetIID),
			TargetTitle:    re.TargetTitle,
			TargetType:     re.TargetType,
		}

		if re.CreatedAt != nil {
			e.CreatedAt = *re.CreatedAt
		}

		if re.PushData.CommitCount > 0 {
			e.PushData = &event.PushData{
				CommitCount: int(re.PushData.CommitCount),
				CommitTitle: re.PushData.CommitTitle,
				CommitTo:    re.PushData.CommitTo,
				Ref:         re.PushData.Ref,
				RefType:     re.PushData.RefType,
			}
		}

		if re.Note != nil {
			if re.Note.Body != "" {
				e.NoteBody = re.Note.Body
			}
			e.NoteableType = re.Note.NoteableType
			e.NoteableIID = int(re.Note.NoteableIID)
		}

		e.ProjectName = c.resolveProject(re.ProjectID)
		events = append(events, e)
	}

	return events, nil
}

// isMergeCommit detects merge commits by their commit title. GitLab generates
// these with a "Merge branch '...' into ..." pattern.
func isMergeCommit(commitTitle string) bool {
	return strings.HasPrefix(commitTitle, "Merge branch '")
}

// CurrentUsername returns the username of the authenticated user by calling
// GET /api/v4/user.
func (c *Client) CurrentUsername() (string, error) {
	user, _, err := c.api.Users.CurrentUser()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

// resolveProject looks up a project's path-with-namespace by its ID, using
// the in-memory cache to avoid redundant API calls. Falls back to
// "project/<id>" if the API call fails.
func (c *Client) resolveProject(id int64) string {
	c.mu.RLock()
	name, ok := c.projectCache[id]
	c.mu.RUnlock()
	if ok {
		return name
	}

	project, _, err := c.api.Projects.GetProject(id, &gl.GetProjectOptions{})
	if err != nil {
		return fmt.Sprintf("project/%d", id)
	}

	c.mu.Lock()
	c.projectCache[id] = project.PathWithNamespace
	c.mu.Unlock()

	return project.PathWithNamespace
}
