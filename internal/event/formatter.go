// Package event defines the domain types and formatting logic for GitLab
// activity events displayed in the TUI.
package event

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// botPattern matches GitLab group/project access token usernames like
// "group_7214_bot_766cc2dcfac8e78d2d1be5b7e06200d7". We strip the trailing
// hash to shorten them to e.g. "group_7214_bot".
var botPattern = regexp.MustCompile(`^((?:group|project)_\d+_bot)_[0-9a-f]{20,}$`)

// authorColors is a narrow palette of blue/teal 256-color ANSI codes with
// enough contrast to distinguish authors while staying in a cohesive hue range.
var authorColors = []string{
	"33",  // dodger blue
	"37",  // cyan
	"38",  // light teal
	"67",  // steel blue
	"68",  // medium blue
	"74",  // sky blue
	"75",  // light steel blue
	"110", // light blue
}

// authorStyles is the pre-computed set of bold lipgloss styles, one per
// entry in authorColors. Built once at init to avoid allocating styles on
// every FormatEvent call.
var authorStyles []lipgloss.Style

func init() {
	authorStyles = make([]lipgloss.Style, len(authorColors))
	for i, c := range authorColors {
		authorStyles[i] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c))
	}
}

// authorStyleFor returns a bold lipgloss style with a deterministic color
// derived from the username. The same username always gets the same color,
// picked from the curated authorColors palette via FNV hash.
func authorStyleFor(username string) lipgloss.Style {
	h := fnv.New32a()
	h.Write([]byte(username))
	return authorStyles[h.Sum32()%uint32(len(authorStyles))]
}

// Shared styles.
var (
	projectStyle   = lipgloss.NewStyle().Faint(true)
	timestampStyle = lipgloss.NewStyle().Faint(true)
	titleStyle     = lipgloss.NewStyle().Faint(true)
)

// Action styles — subtle colors matching the target type palette approach.
var (
	approveAction = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // green
	closeAction   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))   // red
	commentAction = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // yellow
	defaultAction = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))   // white
	mergeAction   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))   // magenta
	openAction    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // green
	pushAction    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // cyan
	deleteAction  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))   // red
)

// Target type styles — subtle but distinct so you can tell at a glance
// what kind of object the event pertains to.
var (
	issueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // green
	milestoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))   // blue
	mrStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	noteStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // yellow
	refStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))   // blue (branch refs)
	snippetStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // cyan
)

// Detail line style — dimmed and indented for the second line of an event.
var detailStyle = lipgloss.NewStyle().Faint(true).Italic(true)

// mentionStyle highlights @username mentions of the current user.
var mentionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")) // yellow bold

// CurrentUser is the authenticated username, set at startup. When non-empty,
// @mentions of this user are highlighted in note bodies.
var CurrentUser string

// ShowFullProject controls whether project names display the full
// path-with-namespace or just the last segment.
var ShowFullProject bool

// FormatEvent renders a single event as one or two lines. The first line shows
// timestamp, author, action, and right-aligned project name. A second detail
// line shows the title for issues/MRs/work items, or a comment snippet for
// comment events. Pass width=0 to skip right-alignment.
func FormatEvent(e Event, width int) string {
	shortName := shortenUsername(e.AuthorUsername)
	author := authorStyleFor(shortName).Render(shortName)
	action := formatAction(e)
	ts := timestampStyle.Render(e.CreatedAt.Local().Format("15:04"))
	project := projectStyle.Render(projectName(e.ProjectName))
	right := project + " " + ts

	left := fmt.Sprintf(" %s %s", author, action)
	if width > 0 {
		leftWidth := lipgloss.Width(left)
		rightWidth := lipgloss.Width(right)
		gap := width - leftWidth - rightWidth - 1
		if gap < 1 {
			gap = 1
		}
		left += strings.Repeat(" ", gap) + right
	} else {
		left += " " + right
	}
	line := left

	// Show a detail line for comments (snippet) and for events with a title
	// (issues, MRs, work items) to keep the first line clean.
	detailMax := 80
	if width > 4 {
		detailMax = width - 4 // leave room for " ↳ " prefix and margin
	}
	if e.NoteBody != "" {
		snippet := truncate(firstLine(e.NoteBody), detailMax)
		line += "\n " + detailStyle.Render("↳ ") + highlightMentions(snippet)
	} else if e.PushData != nil && e.PushData.CommitTitle != "" {
		line += "\n " + detailStyle.Render("↳ "+truncate(e.PushData.CommitTitle, detailMax))
	} else if e.TargetTitle != "" && hasDetailTarget(e.TargetType) {
		line += "\n " + detailStyle.Render("↳ "+truncate(e.TargetTitle, detailMax))
	}

	return line
}

// hasDetailTarget returns true for target types that benefit from showing
// their title on a separate detail line.
func hasDetailTarget(targetType string) bool {
	switch strings.ToLower(targetType) {
	case "issue", "mergerequest", "workitem":
		return true
	}
	return false
}

// highlightMentions renders the raw text with detailStyle, replacing any
// @CurrentUser mentions with mentionStyle highlighting.
func highlightMentions(raw string) string {
	if CurrentUser == "" {
		return detailStyle.Render(raw)
	}
	mention := "@" + CurrentUser
	if !strings.Contains(raw, mention) {
		return detailStyle.Render(raw)
	}
	parts := strings.Split(raw, mention)
	var result []string
	for i, part := range parts {
		result = append(result, detailStyle.Render(part))
		if i < len(parts)-1 {
			result = append(result, mentionStyle.Render(mention))
		}
	}
	return strings.Join(result, "")
}

// projectName returns the display name for a project. When ShowFullProject is
// true, returns the full path-with-namespace; otherwise extracts the last
// segment (e.g. "org/subgroup/project" → "project").
func projectName(name string) string {
	if ShowFullProject {
		return name
	}
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// firstLine returns the first non-empty line of a multi-line string.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return s
}

// truncate shortens a string to maxLen characters, appending "..." if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// shortenUsername truncates long bot usernames by stripping the hash suffix.
// For example "group_7214_bot_766cc2dcfac8e78d2d1be5b7e06200d7" becomes
// "group_7214_bot". Regular usernames are returned as-is.
func shortenUsername(name string) string {
	if m := botPattern.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

// formatAction produces a styled description of the event's action, dispatching
// to specialized formatters based on the action type and presence of push data.
func formatAction(e Event) string {
	switch {
	case e.PushData != nil:
		return formatPush(e)
	case e.ActionName == "opened" || e.ActionName == "created":
		return openAction.Render(e.ActionName) + " " + targetLabel(e)
	case e.ActionName == "closed":
		return closeAction.Render("closed") + " " + targetLabel(e)
	case e.ActionName == "accepted" || e.ActionName == "merged":
		return mergeAction.Render("merged") + " " + targetLabel(e)
	case e.ActionName == "commented on":
		return commentAction.Render("commented on") + " " + targetLabel(e)
	case e.ActionName == "approved":
		return approveAction.Render("approved") + " " + targetLabel(e)
	case e.ActionName == "deleted":
		return deleteAction.Render("deleted") + " " + targetLabel(e)
	default:
		return defaultAction.Render(e.ActionName) + " " + targetLabel(e)
	}
}

func formatPush(e Event) string {
	pd := e.PushData
	commits := "commit"
	if pd.CommitCount != 1 {
		commits = "commits"
	}
	return pushAction.Render(fmt.Sprintf("pushed %d %s", pd.CommitCount, commits)) +
		" to " + refStyle.Render(pd.Ref)
}

func formatPushMultiRef(e Event, refs []string) string {
	pd := e.PushData
	commits := "commit"
	if pd.CommitCount != 1 {
		commits = "commits"
	}
	styledRefs := make([]string, len(refs))
	for i, r := range refs {
		styledRefs[i] = refStyle.Render(r)
	}
	return pushAction.Render(fmt.Sprintf("pushed %d %s", pd.CommitCount, commits)) +
		" to " + strings.Join(styledRefs, ", ")
}

// PushGroupKey returns a grouping key for push events with the same author
// and commit title. Returns empty string for non-push events.
func PushGroupKey(e Event) string {
	if e.PushData == nil || e.PushData.CommitTitle == "" {
		return ""
	}
	return e.AuthorUsername + "\x00" + e.PushData.CommitTitle
}

// FormatGroupedPush renders a push event with multiple branch refs on one line.
func FormatGroupedPush(e Event, refs []string, width int) string {
	shortName := shortenUsername(e.AuthorUsername)
	author := authorStyleFor(shortName).Render(shortName)
	action := formatPushMultiRef(e, refs)
	ts := timestampStyle.Render(e.CreatedAt.Local().Format("15:04"))
	project := projectStyle.Render(projectName(e.ProjectName))
	right := project + " " + ts

	left := fmt.Sprintf(" %s %s", author, action)
	if width > 0 {
		leftWidth := lipgloss.Width(left)
		rightWidth := lipgloss.Width(right)
		gap := width - leftWidth - rightWidth - 1
		if gap < 1 {
			gap = 1
		}
		left += strings.Repeat(" ", gap) + right
	} else {
		left += " " + right
	}
	line := left

	detailMax := 80
	if width > 4 {
		detailMax = width - 4
	}
	if e.PushData.CommitTitle != "" {
		line += "\n " + detailStyle.Render("↳ "+truncate(e.PushData.CommitTitle, detailMax))
	}

	return line
}

// targetLabel builds a human-readable label for the event's target (e.g.
// "issue #42", "MR !7"), with distinct colors per target type. Titles are
// shown inline only for types without a dedicated detail line.
func targetLabel(e Event) string {
	var parts []string

	switch strings.ToLower(e.TargetType) {
	case "issue":
		parts = append(parts, issueStyle.Render(fmt.Sprintf("issue #%d", e.TargetIID)))
	case "mergerequest":
		parts = append(parts, mrStyle.Render(fmt.Sprintf("MR !%d", e.TargetIID)))
	case "milestone":
		parts = append(parts, milestoneStyle.Render("milestone"))
	case "note", "discussionnote":
		parts = append(parts, noteStyle.Render("note"))
	case "snippet":
		parts = append(parts, snippetStyle.Render("snippet"))
	default:
		if e.TargetType != "" {
			parts = append(parts, e.TargetType)
		}
	}

	// Show title inline only for types that don't get a detail line.
	if e.TargetTitle != "" && !hasDetailTarget(e.TargetType) {
		title := truncate(e.TargetTitle, 50)
		parts = append(parts, titleStyle.Render(fmt.Sprintf("%q", title)))
	}

	return strings.Join(parts, " ")
}
