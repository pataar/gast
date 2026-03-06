// Package gitlab provides a client for fetching activity events from a
// GitLab instance via its REST API.
package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	host         string
	token        string
	projectCache map[int]string
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
		host:         strings.TrimRight(host, "/"),
		token:        token,
		projectCache: make(map[int]string),
	}, nil
}

// doGet performs an authenticated GET request to the given URL, returning the
// response. The caller is responsible for closing the response body.
func (c *Client) doGet(reqURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", c.token)
	return http.DefaultClient.Do(req)
}

// rawEvent mirrors the JSON shape returned by GET /events. We decode manually
// because we need to pass the undocumented scope=all parameter which the
// go-gitlab library doesn't support.
type rawEvent struct {
	ID             int    `json:"id"`
	ActionName     string `json:"action_name"`
	AuthorUsername string `json:"author_username"`
	CreatedAt      string `json:"created_at"`
	ProjectID      int    `json:"project_id"`
	TargetType     string `json:"target_type"`
	TargetTitle    string `json:"target_title"`
	TargetIID      int    `json:"target_iid"`
	PushData *struct {
		Action      string `json:"action"`
		CommitCount int    `json:"commit_count"`
		CommitFrom  string `json:"commit_from"`
		CommitTitle string `json:"commit_title"`
		CommitTo    string `json:"commit_to"`
		Ref         string `json:"ref"`
		RefType     string `json:"ref_type"`
	} `json:"push_data"`
	Note *struct {
		Body         string `json:"body"`
		NoteableType string `json:"noteable_type"`
		NoteableIID  int    `json:"noteable_iid"`
	} `json:"note"`
}

// FetchEvents retrieves activity events across all projects the authenticated
// user is a member of (mirroring /dashboard/activity?filter=projects).
// When after is non-nil, only events created after that time are returned.
// Results are sorted in descending order and limited to pageSize entries.
func (c *Client) FetchEvents(after *time.Time, pageSize int) ([]event.Event, error) {
	params := url.Values{
		"sort":     {"desc"},
		"per_page": {strconv.Itoa(pageSize)},
		"page":     {"1"},
		"scope":    {"all"}, // include all project members' activity, not just our own
	}
	if after != nil {
		params.Set("after", after.Format("2006-01-02"))
	}

	reqURL := c.host + "/api/v4/events?" + params.Encode()
	resp, err := c.doGet(reqURL)
	if err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rawEvents []rawEvent
	if err := json.NewDecoder(resp.Body).Decode(&rawEvents); err != nil {
		return nil, fmt.Errorf("decoding events: %w", err)
	}

	events := make([]event.Event, 0, len(rawEvents))
	for _, re := range rawEvents {
		// Skip push events that are merge commits or have no actual commits
		// (e.g. tag pushes, empty pushes after filtering).
		if re.PushData != nil && (isMergeCommit(re.PushData.CommitTitle) || re.PushData.CommitCount == 0) {
			continue
		}

		e := event.Event{
			ID:             re.ID,
			ActionName:     re.ActionName,
			AuthorUsername: re.AuthorUsername,
			TargetIID:      re.TargetIID,
			TargetTitle:    re.TargetTitle,
			TargetType:     re.TargetType,
		}

		// Parse the ISO 8601 timestamp from the API response.
		if t, err := time.Parse(time.RFC3339, re.CreatedAt); err == nil {
			e.CreatedAt = t
		}

		if re.PushData != nil {
			e.PushData = &event.PushData{
				CommitCount: re.PushData.CommitCount,
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
			e.NoteableIID = re.Note.NoteableIID
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
	resp, err := c.doGet(c.host + "/api/v4/user")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.Username, nil
}

// resolveProject looks up a project's path-with-namespace by its ID, using
// the in-memory cache to avoid redundant API calls. Falls back to
// "project/<id>" if the API call fails.
func (c *Client) resolveProject(id int) string {
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
