// Package event defines the domain types and formatting logic for GitLab activity events shown in the TUI.
package event

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// botPattern matches bot usernames like "group_7214_bot_<hex hash>"; shortenUsername strips the hash suffix.
var botPattern = regexp.MustCompile(`^((?:group|project)_\d+_bot)_[0-9a-f]{20,}$`)

// authorColors is a narrow palette of blue/teal 256-color ANSI codes — distinct authors, cohesive hue range.
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

// authorStyles holds one pre-built bold style per authorColors entry, built once to avoid per-render allocations.
var authorStyles []lipgloss.Style

func init() {
	authorStyles = make([]lipgloss.Style, len(authorColors))
	for i, c := range authorColors {
		authorStyles[i] = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c))
	}
}

/*
authorStyleFor picks a deterministic style per username via inlined FNV-1a (offset basis
2166136261, prime 16777619), avoiding the per-call hasher and []byte allocations in the render path.
*/
func authorStyleFor(username string) lipgloss.Style {
	h := uint32(2166136261)
	for i := 0; i < len(username); i++ {
		h ^= uint32(username[i])
		h *= 16777619
	}
	return authorStyles[h%uint32(len(authorStyles))]
}

// Shared styles.
var (
	bracketStyle   = lipgloss.NewStyle().Faint(true)
	projectStyle   = lipgloss.NewStyle().Faint(true)
	timestampStyle = lipgloss.NewStyle().Faint(true)
	titleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

// Action styles — subtle colors matching the target type palette approach.
var (
	approveAction = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	closeAction   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	commentAction = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	defaultAction = lipgloss.NewStyle().Foreground(lipgloss.Color("7")) // white
	mergeAction   = lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // magenta
	openAction    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	pushAction    = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	deleteAction  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
)

// Target type styles — subtle but distinct so the object kind is clear at a glance.
var (
	issueStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // green
	milestoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))   // blue
	mrStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	noteStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // yellow
	refStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))   // blue (branch refs)
	snippetStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // cyan
)

// Detail line style — lighter gray for the second line of an event.
var detailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Italic(true)

// mentionStyle highlights @username mentions of the current user.
var mentionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")) // yellow bold

// CurrentUser is the authenticated username, set at startup; @mentions of it are highlighted in note bodies.
var CurrentUser string

// ShowFullProject switches project names between full path-with-namespace and just the last segment.
var ShowFullProject bool

// RelativeTime toggles relative ("3m ago") vs absolute ("15:04") timestamps; flipped at runtime with 't'.
var RelativeTime bool

/*
FormatEvent renders a single event as one or two lines. The first line shows timestamp, author,
action, and right-aligned project name. A second detail line shows the title for issues/MRs/work
items, or a comment snippet for comment events. Pass width=0 to skip right-alignment.
*/
func FormatEvent(e Event, width int) string {
	return formatLine(e, formatAction(e), width)
}

// FormatGroupedPush renders a push event with multiple branch refs on one line.
func FormatGroupedPush(e Event, refs []string, width int) string {
	return formatLine(e, formatPush(e, refs), width)
}

// formatLine assembles the shared event layout for the given action, plus an optional detail line.
func formatLine(e Event, action string, width int) string {
	shortName := shortenUsername(e.AuthorUsername)
	author := authorStyleFor(shortName).Render(shortName)
	ts := timestampStyle.Render(formatTimestamp(e.CreatedAt))
	project := projectStyle.Render(projectName(e.ProjectName))
	right := bracketStyle.Render("[") + project + " " + ts + bracketStyle.Render("]")

	left := fmt.Sprintf(" %s %s", author, action)
	line := AlignLeftRight(left, right, width)

	detailMax := 80
	if width > 4 {
		detailMax = width - 4 // leave room for " ↳ " prefix and margin
	}
	if detail := formatDetail(e, detailMax); detail != "" {
		line += "\n " + detail
	}

	return line
}

// formatDetail renders the detail line (comment snippet, push commit title, or target title), or "" when none.
func formatDetail(e Event, detailMax int) string {
	switch {
	case e.NoteBody != "":
		snippet := Truncate(firstLine(e.NoteBody), detailMax)
		return detailStyle.Render("↳ ") + highlightMentions(snippet)
	case e.PushData != nil && e.PushData.CommitTitle != "":
		return detailStyle.Render("↳ " + Truncate(e.PushData.CommitTitle, detailMax))
	case e.TargetTitle != "" && hasDetailTarget(e.TargetType):
		return detailStyle.Render("↳ " + Truncate(e.TargetTitle, detailMax))
	default:
		return ""
	}
}

// HasDetailLine reports whether formatting e yields a second detail line; mirrors formatDetail's cases.
func HasDetailLine(e Event) bool {
	return e.NoteBody != "" ||
		(e.PushData != nil && e.PushData.CommitTitle != "") ||
		(e.TargetTitle != "" && hasDetailTarget(e.TargetType))
}

// hasDetailTarget returns true for target types whose title is shown on a dedicated detail line.
func hasDetailTarget(targetType string) bool {
	return IsIssueTargetType(targetType) || strings.EqualFold(targetType, "mergerequest")
}

// HasMention reports whether the text mentions the current user.
func HasMention(text string) bool {
	return CurrentUser != "" && strings.Contains(text, "@"+CurrentUser)
}

// highlightMentions renders raw text with detailStyle, highlighting @CurrentUser mentions with mentionStyle.
func highlightMentions(raw string) string {
	if !HasMention(raw) {
		return detailStyle.Render(raw)
	}
	mention := "@" + CurrentUser
	parts := strings.Split(raw, mention)
	var b strings.Builder
	for i, part := range parts {
		b.WriteString(detailStyle.Render(part))
		if i < len(parts)-1 {
			b.WriteString(mentionStyle.Render(mention))
		}
	}
	return b.String()
}

// projectName returns the full path-with-namespace when ShowFullProject is set, else the last path segment.
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

// formatTimestamp renders t relative or absolute depending on the RelativeTime setting.
func formatTimestamp(t time.Time) string {
	if !RelativeTime {
		return t.Local().Format("15:04")
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

/*
AlignLeftRight places left and right content on a single line, padding to right-align the right
portion. An overlong left is truncated (ANSI-aware) to preserve the right portion plus a gap.
*/
func AlignLeftRight(left, right string, width int) string {
	if width <= 0 {
		return left + " " + right
	}
	rightWidth := lipgloss.Width(right)
	maxLeft := width - rightWidth - 2 // 2 = minimum gap
	leftWidth := lipgloss.Width(left)
	if leftWidth > maxLeft && maxLeft > 3 {
		left = lipgloss.NewStyle().MaxWidth(maxLeft).Render(left)
		leftWidth = lipgloss.Width(left)
	}
	gap := width - leftWidth - rightWidth - 1
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// Truncate shortens s to maxLen visible runes, appending "..." — rune-counted so multi-byte UTF-8 measures correctly.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s // byte length bounds rune count, so no cut is needed
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// shortenUsername strips the hash suffix from bot usernames ("group_7214_bot_766cc…" → "group_7214_bot").
func shortenUsername(name string) string {
	// Cheap prefix check so the regex only runs for candidate bot names.
	if !strings.HasPrefix(name, "group_") && !strings.HasPrefix(name, "project_") {
		return name
	}
	if m := botPattern.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

// formatAction produces a styled description of the event's action, dispatching pushes to formatPush.
func formatAction(e Event) string {
	if e.PushData != nil {
		return formatPush(e, []string{e.PushData.Ref})
	}

	style, verb := defaultAction, e.ActionName
	switch e.ActionName {
	case "opened", "created":
		style = openAction
	case "closed":
		style = closeAction
	case "accepted", "merged":
		style, verb = mergeAction, "merged"
	case "commented on":
		style = commentAction
	case "approved":
		style = approveAction
	case "deleted":
		style = deleteAction
	}
	return style.Render(verb) + " " + targetLabel(e)
}

// formatPush renders a push action for one or more branch refs.
func formatPush(e Event, refs []string) string {
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

/*
PushGroupKey returns a grouping key for push events sharing an author and commit title.
The bool is false for events that cannot be grouped (non-pushes, pushes without a commit title).
*/
func PushGroupKey(e Event) (string, bool) {
	if e.PushData == nil || e.PushData.CommitTitle == "" {
		return "", false
	}
	return e.AuthorUsername + "\x00" + e.ProjectName + "\x00" + e.PushData.CommitTitle, true
}

/*
targetLabel builds a colored label for the event's target (e.g. "issue #42", "MR !7").
Titles are shown inline only for types without a dedicated detail line.
*/
func targetLabel(e Event) string {
	var label string
	switch {
	case IsIssueTargetType(e.TargetType):
		label = issueStyle.Render(fmt.Sprintf("issue #%d", e.TargetIID))
	case strings.EqualFold(e.TargetType, "mergerequest"):
		label = mrStyle.Render(fmt.Sprintf("MR !%d", e.TargetIID))
	case strings.EqualFold(e.TargetType, "milestone"):
		label = milestoneStyle.Render("milestone")
	case IsNoteTargetType(e.TargetType):
		label = noteStyle.Render("note")
	case strings.EqualFold(e.TargetType, "snippet"):
		label = snippetStyle.Render("snippet")
	default:
		label = e.TargetType
	}

	// Inline title only for types without a detail line; AlignLeftRight handles overflow.
	if e.TargetTitle != "" && !hasDetailTarget(e.TargetType) {
		title := titleStyle.Render(fmt.Sprintf("%q", e.TargetTitle))
		if label == "" {
			return title
		}
		return label + " " + title
	}
	return label
}
