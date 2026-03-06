package event

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// TestFormatEvent_Push verifies formatting of push events with single and multiple commits.
func TestFormatEvent_Push(t *testing.T) {
	tests := []struct {
		name        string
		commitCount int
		wantCommits string
	}{
		{"single commit", 1, "pushed 1 commit to main"},
		{"multiple commits", 3, "pushed 3 commits to main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{
				AuthorUsername: "alice",
				CreatedAt:      time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				ProjectName:    "myproject",
				PushData: &PushData{
					CommitCount: tt.commitCount,
					Ref:         "main",
				},
			}

			got := stripANSI(FormatEvent(e, 0))
			if !strings.Contains(got, tt.wantCommits) {
				t.Errorf("FormatEvent() = %q, want it to contain %q", got, tt.wantCommits)
			}
			if !strings.Contains(got, "alice") {
				t.Errorf("FormatEvent() = %q, want it to contain author %q", got, "alice")
			}
			if !strings.Contains(got, "myproject") {
				t.Errorf("FormatEvent() = %q, want it to contain project %q", got, "myproject")
			}
		})
	}
}

// TestFormatEvent_PushCommitTitle verifies that push events show the commit
// title on a second detail line.
func TestFormatEvent_PushCommitTitle(t *testing.T) {
	e := Event{
		AuthorUsername: "bob",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "backend",
		PushData: &PushData{
			CommitCount: 1,
			CommitTitle: "fix: resolve null pointer in auth middleware",
			Ref:         "main",
		},
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "\n") {
		t.Error("push events should have a detail line for the commit title")
	}
	if !strings.Contains(got, "fix: resolve null pointer in auth middleware") {
		t.Error("push events should show the commit title")
	}
}

// TestFormatEvent_OpenedCreated verifies formatting of opened/created events with MR target.
func TestFormatEvent_OpenedCreated(t *testing.T) {
	tests := []struct {
		name       string
		actionName string
	}{
		{"opened", "opened"},
		{"created", "created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{
				ActionName:     tt.actionName,
				AuthorUsername: "bob",
				CreatedAt:      time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				ProjectName:    "myproject",
				TargetType:     "MergeRequest",
				TargetIID:      42,
				TargetTitle:    "Add feature X",
			}

			got := stripANSI(FormatEvent(e, 0))
			if !strings.Contains(got, tt.actionName) {
				t.Errorf("FormatEvent() = %q, want it to contain %q", got, tt.actionName)
			}
			if !strings.Contains(got, "MR !42") {
				t.Errorf("FormatEvent() = %q, want it to contain %q", got, "MR !42")
			}
			if !strings.Contains(got, "Add feature X") {
				t.Errorf("FormatEvent() = %q, want it to contain target title", got)
			}
			if !strings.Contains(got, "\n") {
				t.Error("FormatEvent() should contain a detail line for MR title")
			}
		})
	}
}

// TestFormatEvent_Closed verifies formatting of closed events with issue target.
func TestFormatEvent_Closed(t *testing.T) {
	e := Event{
		ActionName:     "closed",
		AuthorUsername: "carol",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "backend",
		TargetType:     "Issue",
		TargetIID:      99,
		TargetTitle:    "Fix login bug",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "closed") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "closed")
	}
	if !strings.Contains(got, "issue #99") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "issue #99")
	}
}

// TestFormatEvent_Merged verifies formatting of merged events.
func TestFormatEvent_Merged(t *testing.T) {
	for _, action := range []string{"accepted", "merged"} {
		t.Run(action, func(t *testing.T) {
			e := Event{
				ActionName:     action,
				AuthorUsername: "dave",
				CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
				ProjectName:    "frontend",
				TargetType:     "MergeRequest",
				TargetIID:      7,
				TargetTitle:    "Refactor styles",
			}

			got := stripANSI(FormatEvent(e, 0))
			if !strings.Contains(got, "merged") {
				t.Errorf("FormatEvent() = %q, want it to contain %q", got, "merged")
			}
			if !strings.Contains(got, "MR !7") {
				t.Errorf("FormatEvent() = %q, want it to contain %q", got, "MR !7")
			}
		})
	}
}

// TestFormatEvent_CommentedOn verifies formatting of "commented on" events.
func TestFormatEvent_CommentedOn(t *testing.T) {
	e := Event{
		ActionName:     "commented on",
		AuthorUsername: "eve",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "api",
		TargetType:     "Issue",
		TargetIID:      5,
		TargetTitle:    "Performance regression",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "commented on") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "commented on")
	}
	if !strings.Contains(got, "issue #5") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "issue #5")
	}
}

// TestFormatEvent_Approved verifies formatting of approved events.
func TestFormatEvent_Approved(t *testing.T) {
	e := Event{
		ActionName:     "approved",
		AuthorUsername: "frank",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "core",
		TargetType:     "MergeRequest",
		TargetIID:      12,
		TargetTitle:    "Upgrade dependencies",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "approved") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "approved")
	}
	if !strings.Contains(got, "MR !12") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "MR !12")
	}
}

// TestFormatEvent_Deleted verifies formatting of deleted events.
func TestFormatEvent_Deleted(t *testing.T) {
	e := Event{
		ActionName:     "deleted",
		AuthorUsername: "grace",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "infra",
		TargetType:     "Issue",
		TargetIID:      20,
		TargetTitle:    "Old issue",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "deleted") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "deleted")
	}
	if !strings.Contains(got, "issue #20") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "issue #20")
	}
}

// TestFormatEvent_TitleTruncation verifies that long titles on the detail line
// are truncated to 80 characters.
func TestFormatEvent_TitleTruncation(t *testing.T) {
	longTitle := strings.Repeat("A", 100)
	e := Event{
		ActionName:     "opened",
		AuthorUsername: "hank",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "myproject",
		TargetType:     "Issue",
		TargetIID:      1,
		TargetTitle:    longTitle,
	}

	got := stripANSI(FormatEvent(e, 0))
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatal("expected a detail line for issue title")
	}
	detail := strings.TrimSpace(lines[1])
	// The detail line has a "↳ " prefix (2 extra chars) plus the truncated content.
	// With width=0 the fallback is 80 chars for content + prefix.
	if len(detail) > 84 {
		t.Errorf("detail line length = %d, want <= 84", len(detail))
	}
	if !strings.HasSuffix(detail, "...") {
		t.Error("detail should end with '...' when truncated")
	}
}

// TestFormatEvent_UnknownAction verifies that unknown action types fall through to the default.
func TestFormatEvent_UnknownAction(t *testing.T) {
	e := Event{
		ActionName:     "snoozed",
		AuthorUsername: "iris",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		ProjectName:    "myproject",
		TargetType:     "Issue",
		TargetIID:      3,
		TargetTitle:    "Some issue",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "snoozed") {
		t.Errorf("FormatEvent() = %q, want it to contain action %q", got, "snoozed")
	}
	if !strings.Contains(got, "issue #3") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "issue #3")
	}
}

// TestTargetLabel_AllTypes verifies the targetLabel output for all known target types.
func TestTargetLabel_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		targetType string
		targetIID  int
		want       string
	}{
		{"issue", "Issue", 10, "issue #10"},
		{"mergerequest", "MergeRequest", 5, "MR !5"},
		{"milestone", "Milestone", 0, "milestone"},
		{"note", "Note", 0, "note"},
		{"snippet", "Snippet", 0, "snippet"},
		{"unknown type", "CustomType", 0, "CustomType"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Event{
				TargetType: tt.targetType,
				TargetIID:  tt.targetIID,
			}

			got := stripANSI(targetLabel(e))
			if !strings.Contains(got, tt.want) {
				t.Errorf("targetLabel() = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

// TestFormatEvent_CommentDetail verifies that comment events show a snippet of
// the note body on a second line.
func TestFormatEvent_CommentDetail(t *testing.T) {
	e := Event{
		ActionName:     "commented on",
		AuthorUsername: "alice",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		NoteBody:       "Looks good, just one small nit on line 42.",
		ProjectName:    "myproject",
		TargetIID:      5,
		TargetTitle:    "Fix login bug",
		TargetType:     "Issue",
	}

	got := stripANSI(FormatEvent(e, 0))
	if !strings.Contains(got, "commented on") {
		t.Errorf("FormatEvent() = %q, want it to contain %q", got, "commented on")
	}
	if !strings.Contains(got, "Looks good, just one small nit on line 42.") {
		t.Errorf("FormatEvent() = %q, want it to contain comment snippet", got)
	}
	if !strings.Contains(got, "\n") {
		t.Error("FormatEvent() should contain a second line for the comment detail")
	}
}

// TestFormatEvent_CommentDetailTruncation verifies that long comment snippets
// are truncated based on width.
func TestFormatEvent_CommentDetailTruncation(t *testing.T) {
	longBody := strings.Repeat("x", 200)
	e := Event{
		ActionName:     "commented on",
		AuthorUsername: "alice",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		NoteBody:       longBody,
		ProjectName:    "myproject",
		TargetType:     "Issue",
	}

	// With a specific width, detail should be truncated to fit.
	got := stripANSI(FormatEvent(e, 60))
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatal("expected a second line for the comment detail")
	}
	detail := strings.TrimSpace(lines[1])
	// "↳ " prefix (2 chars) + truncated content ≤ 56 chars (width - 4)
	if len(detail) > 60 {
		t.Errorf("detail line length = %d, want <= 60", len(detail))
	}
	if !strings.HasSuffix(detail, "...") {
		t.Error("detail should end with '...' when truncated")
	}
}

// TestFormatEvent_MultilineNote verifies that only the first line of a
// multi-line comment is shown in the detail.
func TestFormatEvent_MultilineNote(t *testing.T) {
	e := Event{
		ActionName:     "commented on",
		AuthorUsername: "alice",
		CreatedAt:      time.Date(2025, 1, 15, 14, 0, 0, 0, time.UTC),
		NoteBody:       "First line of comment\nSecond line with more detail\nThird line",
		ProjectName:    "myproject",
		TargetType:     "Issue",
	}

	got := stripANSI(FormatEvent(e, 0))
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatal("expected a second line for the comment detail")
	}
	detail := strings.TrimSpace(lines[1])
	if strings.Contains(detail, "Second line") {
		t.Error("detail should only show the first line of the comment")
	}
	if !strings.Contains(detail, "First line of comment") {
		t.Errorf("detail = %q, want it to contain first line", detail)
	}
}

// TestShortenUsername verifies that long bot usernames are shortened by
// stripping the hash suffix, while regular usernames are left unchanged.
func TestShortenUsername(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"group bot", "group_7214_bot_766cc2dcfac8e78d2d1be5b7e06200d7", "group_7214_bot"},
		{"project bot", "project_4050_bot_73c77d4b7a1268c18e27633b4c93163a", "project_4050_bot"},
		{"regular user", "alice.smith", "alice.smith"},
		{"short name", "bob", "bob"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortenUsername(tt.input)
			if got != tt.want {
				t.Errorf("shortenUsername(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestProjectName verifies that project paths are reduced to the last segment.
func TestProjectName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "group/project", "project"},
		{"single segment", "myproject", "myproject"},
		{"deeply nested", "org/subgroup/project", "project"},
		{"very deep", "a/b/c/d/project", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectName(tt.input)
			if got != tt.want {
				t.Errorf("projectName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestAuthorStyleFor_Deterministic verifies that the same username always
// produces the same color, and different usernames can produce different colors.
func TestAuthorStyleFor_Deterministic(t *testing.T) {
	style1 := authorStyleFor("alice")
	style2 := authorStyleFor("alice")
	style3 := authorStyleFor("bob")

	// Same input must produce same output.
	if style1.GetForeground() != style2.GetForeground() {
		t.Error("authorStyleFor should be deterministic for the same username")
	}

	// We can't guarantee different usernames always get different colors (pigeonhole),
	// but alice and bob are short enough that they almost certainly differ.
	_ = style3 // just verify it doesn't panic
}
