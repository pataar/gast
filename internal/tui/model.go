// Package tui implements the Bubble Tea model, view, and update logic for the
// GitLab activity stream terminal user interface.
package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/pataar/gast/internal/browser"
	"github.com/pataar/gast/internal/config"
	"github.com/pataar/gast/internal/event"
	"github.com/pataar/gast/internal/gitlab"
	"github.com/pataar/gast/internal/notify"
)

// maxEvents is the upper bound on events kept in memory. Older events beyond
// this limit are discarded to prevent unbounded memory growth.
const maxEvents = 500

// displayItem represents a visual item in the event list. A single item may
// represent one event or a group of push events to the same commit.
type displayItem struct {
	primaryEvent event.Event // the representative event for this item
	groupedRefs  []string    // branch refs for grouped push events (len > 1 means grouped)
}

// Model is the Bubble Tea model that manages application state including the
// event list, viewport, spinner, and polling lifecycle.
type Model struct {
	client   *gitlab.Client
	cfg      *config.Config
	events   []event.Event
	seenIDs  map[int]struct{}
	viewport viewport.Model
	spinner  spinner.Model
	keys     KeyMap
	width    int
	height   int

	fetching        bool
	initialFetch    bool // true until the first successful fetch completes
	manualRefresh   bool // true when user pressed 'r'
	lastUpdate      time.Time
	err             error
	showHelp        bool
	initialized     bool
	demo            bool
	demoEvents      []event.Event
	consecutiveErrs int
	clearedAt       *time.Time // if set, ignore events before this time
	projectFilters  []string   // filter events to these project path substrings
	groupFilters    []string   // filter events to these group path prefixes
	displayItems    []displayItem
	selectedIdx     int
	mentionCount    int // unread @mention count
}

// NewModel creates a new TUI model wired to the given GitLab client and config.
func NewModel(client *gitlab.Client, cfg *config.Config) Model {
	s := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(spinnerStyle))

	// Set display preferences from config.
	event.CurrentUser = cfg.Username
	event.ShowFullProject = cfg.ShowFullProject

	return Model{
		client:       client,
		cfg:          cfg,
		seenIDs:      make(map[int]struct{}),
		spinner:      s,
		keys:         defaultKeyMap(),
		initialFetch: true,
	}
}

// NewDemoModel creates a TUI model pre-loaded with fake events (no GitLab client).
func NewDemoModel(cfg *config.Config, events []event.Event) Model {
	m := NewModel(nil, cfg)
	m.demo = true
	m.demoEvents = events
	return m
}

// Init starts the spinner animation, triggers the first event fetch, and
// schedules the first polling tick.
func (m Model) Init() tea.Cmd {
	if m.demo {
		return func() tea.Msg {
			return EventsFetchedMsg{Events: m.demoEvents}
		}
	}
	return tea.Batch(
		m.spinner.Tick,
		fetchEventsCmd(m.client, nil, m.cfg.PageSize),
	)
}

// Update handles incoming messages (key presses, window resizes, fetch results,
// and timer ticks) and returns the updated model and any follow-up commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 2
		footerHeight := 2
		vpHeight := m.height - headerHeight - footerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.initialized {
			m.viewport = viewport.New(viewport.WithWidth(m.width), viewport.WithHeight(vpHeight))
			m.initialized = true
		} else {
			m.viewport.SetWidth(m.width)
			m.viewport.SetHeight(vpHeight)
		}
		m.viewport.SetContent(m.renderEvents())

	case EventsFetchedMsg:
		m.fetching = false
		m.initialFetch = false
		m.manualRefresh = false
		m.lastUpdate = time.Now()
		m.err = nil
		m.consecutiveErrs = 0
		m.checkMentions(msg.Events)
		m.mergeEvents(msg.Events)
		oldItemCount := len(m.displayItems)
		m.buildDisplayItems()
		if m.initialized {
			// Only auto-scroll to bottom if user was already at the end
			// or this is the first fetch.
			wasAtEnd := m.selectedIdx >= oldItemCount-1 || oldItemCount == 0
			if wasAtEnd && len(m.displayItems) > 0 {
				m.selectedIdx = len(m.displayItems) - 1
			}
			m.viewport.SetContent(m.renderEvents())
			if wasAtEnd {
				m.viewport.GotoBottom()
			}
		} else if len(m.displayItems) > 0 {
			m.selectedIdx = len(m.displayItems) - 1
		}
		// Lazily resolve truncated commit titles in the background.
		cmds = append(cmds, m.resolveCommitTitles(msg.Events))
		// Schedule the next poll after a successful fetch (skip in demo mode).
		if !m.demo {
			cmds = append(cmds, tickCmd(m.cfg.PollInterval))
		}

	case FetchErrorMsg:
		m.fetching = false
		m.err = msg.Err
		m.consecutiveErrs++
		cmds = append(cmds, tickCmd(m.backoffInterval()))

	case CommitTitleMsg:
		// Update the commit title on the matching event and rebuild display.
		for i := range m.events {
			if m.events[i].ID == msg.EventID && m.events[i].PushData != nil {
				m.events[i].PushData.CommitTitle = msg.Title
				break
			}
		}
		m.buildDisplayItems()
		if m.initialized {
			m.viewport.SetContent(m.renderEvents())
		}

	case TickMsg:
		m.fetching = true
		cmds = append(cmds, m.spinner.Tick, m.fetchCmd())

	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Quit) || msg.String() == "esc" {
				m.showHelp = false
				m.viewport.SetContent(m.renderEvents())
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = true
			m.viewport.SetContent(m.renderHelp())
			return m, nil
		case key.Matches(msg, m.keys.Open):
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.displayItems) {
				e := m.displayItems[m.selectedIdx].primaryEvent
				if host := m.gitlabHost(); host != "" {
					_ = browser.OpenEvent(host, e)
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.OpenProject):
			if m.selectedIdx >= 0 && m.selectedIdx < len(m.displayItems) {
				e := m.displayItems[m.selectedIdx].primaryEvent
				if host := m.gitlabHost(); host != "" && e.ProjectName != "" {
					_ = browser.Open(fmt.Sprintf("%s/%s", strings.TrimRight(host, "/"), e.ProjectName))
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Refresh):
			m.fetching = true
			m.manualRefresh = true
			return m, tea.Batch(m.spinner.Tick, m.fetchCmd())
		case key.Matches(msg, m.keys.Clear):
			now := time.Now()
			m.clearedAt = &now
			m.events = m.events[:0]
			m.seenIDs = make(map[int]struct{})
			m.displayItems = m.displayItems[:0]
			m.selectedIdx = 0
			m.mentionCount = 0
			if m.initialized {
				m.viewport.SetContent(m.renderEvents())
				m.viewport.GotoTop()
			}
			return m, nil
		case key.Matches(msg, m.keys.ToggleTime):
			event.RelativeTime = !event.RelativeTime
			if m.initialized {
				m.viewport.SetContent(m.renderEvents())
			}
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.mentionCount = 0
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.viewport.SetContent(m.renderEvents())
				m.scrollToSelected()
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.mentionCount = 0
			if m.selectedIdx < len(m.displayItems)-1 {
				m.selectedIdx++
				m.viewport.SetContent(m.renderEvents())
				m.scrollToSelected()
			}
			return m, nil
		case key.Matches(msg, m.keys.GoTop):
			m.selectedIdx = 0
			m.viewport.SetContent(m.renderEvents())
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.GoBottom):
			if len(m.displayItems) > 0 {
				m.selectedIdx = len(m.displayItems) - 1
			}
			m.viewport.SetContent(m.renderEvents())
			m.viewport.GotoBottom()
			return m, nil
		}

	case spinner.TickMsg:
		// Only animate the spinner during initial/manual fetch to avoid
		// unnecessary re-renders (which cause visible flashing).
		if m.fetching && (m.initialFetch || m.manualRefresh) {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Forward non-key messages to the viewport (window size, mouse wheel, etc.)
	// but not key messages — we handle those ourselves for event selection.
	if m.initialized && !m.showHelp {
		switch msg.(type) {
		case tea.KeyMsg:
			// Handled above via key bindings — don't forward to viewport.
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the full TUI layout: header, divider, scrollable event list,
// divider, and footer. Before initialization, it shows a loading spinner.
func (m Model) View() tea.View {
	if !m.initialized {
		v := tea.NewView(fmt.Sprintf("\n %s Loading...\n", m.spinner.View()))
		v.AltScreen = true
		return v
	}

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(m.renderDivider())
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) renderHeader() string {
	title := headerStyle.Render("GitLab Activity Stream")
	if m.mentionCount > 0 {
		title += " " + mentionBadgeStyle.Render(fmt.Sprintf(" @%d ", m.mentionCount))
	}

	right := ""
	if m.fetching && (m.initialFetch || m.manualRefresh) {
		right = m.spinner.View() + " Fetching..."
	} else if !m.lastUpdate.IsZero() {
		right = fmt.Sprintf("Last updated: %s    ↻ %s",
			m.lastUpdate.Local().Format("15:04:05"),
			m.cfg.PollInterval)
	}

	spaces := m.width - lipgloss.Width(title) - lipgloss.Width(right) - 1
	if spaces < 1 {
		spaces = 1
	}

	return title + strings.Repeat(" ", spaces) + right
}

func (m Model) renderDivider() string {
	return dividerStyle.Render(strings.Repeat("─", m.width))
}

func (m Model) renderFooter() string {
	left := " j/k select  o open  p project  r refresh  c clear  t time  ? help  q quit"

	eventCount := fmt.Sprintf("%d events", len(m.events))
	if m.err != nil {
		eventCount = errorStyle.Render(fmt.Sprintf("error: %v", m.err))
	}

	spaces := m.width - lipgloss.Width(left) - lipgloss.Width(eventCount) - 1
	if spaces < 1 {
		spaces = 1
	}

	return footerStyle.Render(left + strings.Repeat(" ", spaces) + eventCount)
}

func (m Model) renderEvents() string {
	if len(m.displayItems) == 0 {
		if m.clearedAt != nil {
			return "\n  No events yet. Waiting for new events..."
		}
		return "\n  No events yet. Waiting for first fetch..."
	}

	// Leave 2 chars for the selection indicator prefix.
	contentWidth := m.width - 2
	if contentWidth < 10 {
		contentWidth = 10
	}

	var blocks []string
	for i, item := range m.displayItems {
		var block string
		if len(item.groupedRefs) > 1 {
			block = event.FormatGroupedPush(item.primaryEvent, item.groupedRefs, contentWidth)
		} else {
			block = event.FormatEvent(item.primaryEvent, contentWidth)
		}

		// Add selection indicator.
		prefix := "  "
		if i == m.selectedIdx {
			prefix = selectedIndicatorStyle.Render("▸ ")
		}
		lines := strings.Split(block, "\n")
		for j, line := range lines {
			if j == 0 {
				lines[j] = prefix + line
			} else {
				lines[j] = "  " + line
			}
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	sep := dividerStyle.Render(strings.Repeat("┄", m.width))
	return strings.Join(blocks, "\n"+sep+"\n")
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(helpTitleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := []struct{ key, desc string }{
		{"j / down", "Select next event"},
		{"k / up", "Select previous event"},
		{"g / Home", "Select first event"},
		{"G / End", "Select last event"},
		{"o / Enter", "Open event in browser"},
		{"p", "Open project in browser"},
		{"r", "Force refresh"},
		{"c", "Clear events"},
		{"t", "Toggle relative/absolute time"},
		{"?", "Toggle this help"},
		{"q / Ctrl+C", "Quit"},
	}

	for _, bind := range bindings {
		line := fmt.Sprintf("  %-14s %s", bind.key, bind.desc)
		b.WriteString(helpStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// scrollToSelected adjusts the viewport offset to keep the selected item visible.
func (m *Model) scrollToSelected() {
	// Calculate the line offset of the selected item by counting lines of
	// all preceding items plus separators.
	lineOffset := 0
	for i := 0; i < m.selectedIdx && i < len(m.displayItems); i++ {
		lineOffset += m.itemLineCount(i)
		lineOffset++ // separator line between items
	}
	selectedLines := m.itemLineCount(m.selectedIdx)

	// If the selected item is above the viewport, scroll up to it.
	if lineOffset < m.viewport.YOffset() {
		m.viewport.SetYOffset(lineOffset)
	}
	// If the selected item is below the viewport, scroll down so it's visible.
	vpHeight := m.viewport.Height()
	if lineOffset+selectedLines > m.viewport.YOffset()+vpHeight {
		m.viewport.SetYOffset(lineOffset + selectedLines - vpHeight)
	}
}

// resolveCommitTitles returns commands to fetch full titles for any push events
// with truncated commit titles (ending in "..."). Called after events are fetched.
func (m Model) resolveCommitTitles(newEvents []event.Event) tea.Cmd {
	if m.client == nil {
		return nil
	}
	var cmds []tea.Cmd
	for _, e := range newEvents {
		if e.PushData == nil || e.PushData.CommitTo == "" {
			continue
		}
		if !strings.HasSuffix(e.PushData.CommitTitle, "...") {
			continue
		}
		cmds = append(cmds, resolveCommitTitleCmd(m.client, e.ID, e.ProjectID, e.PushData.CommitTo, e.PushData.CommitTitle))
	}
	return tea.Batch(cmds...)
}

// itemLineCount returns the number of rendered lines for a display item.
func (m Model) itemLineCount(idx int) int {
	if idx < 0 || idx >= len(m.displayItems) {
		return 1
	}
	item := m.displayItems[idx]
	e := item.primaryEvent
	lines := 1
	if e.NoteBody != "" || (e.PushData != nil && e.PushData.CommitTitle != "") ||
		(e.TargetTitle != "" && event.HasDetailTarget(e.TargetType)) {
		lines = 2
	}
	return lines
}

// gitlabHost returns the configured GitLab host URL, or empty string.
func (m Model) gitlabHost() string {
	if m.cfg != nil {
		return m.cfg.GitLabHost
	}
	return ""
}

// backoffInterval returns the retry interval with exponential backoff based on
// consecutive error count. Caps at 5 minutes.
func (m Model) backoffInterval() time.Duration {
	base := m.cfg.PollInterval
	for i := 0; i < m.consecutiveErrs-1; i++ {
		base *= 2
		if base > 5*time.Minute {
			return 5 * time.Minute
		}
	}
	return base
}

// fetchCmd builds the fetch command using the latest event's timestamp as the
// "after" filter, or nil for the initial fetch.
func (m Model) fetchCmd() tea.Cmd {
	var after *time.Time
	if len(m.events) > 0 {
		// GitLab's "after" param is date-only (YYYY-MM-DD) and exclusive,
		// so subtract a day to ensure same-day events are still returned.
		// Duplicates are filtered out by seenIDs in mergeEvents.
		t := m.events[len(m.events)-1].CreatedAt.Add(-24 * time.Hour)
		after = &t
	} else if m.clearedAt != nil {
		// After a clear, use the clear timestamp so we don't re-fetch
		// old events.
		t := m.clearedAt.Add(-24 * time.Hour)
		after = &t
	}
	return fetchEventsCmd(m.client, after, m.cfg.PageSize)
}

// checkMentions scans new events for @mentions of the current user. When found,
// increments the mention counter and optionally sends a desktop notification.
func (m *Model) checkMentions(newEvents []event.Event) {
	if event.CurrentUser == "" {
		return
	}
	mention := "@" + event.CurrentUser
	for _, e := range newEvents {
		if _, seen := m.seenIDs[e.ID]; seen {
			continue
		}
		if e.AuthorUsername == event.CurrentUser {
			continue
		}
		if !strings.Contains(e.NoteBody, mention) {
			continue
		}
		m.mentionCount++
		if m.cfg != nil && m.cfg.Notifications {
			body := notify.FormatMention(e.AuthorUsername, e.ProjectName, e.NoteBody)
			url := ""
			if host := m.gitlabHost(); host != "" {
				url = browser.EventURL(host, e)
			}
			_ = notify.Send("gast — @mention", body, url)
		}
	}
}

// buildDisplayItems creates the list of visual display items from the raw
// event list, grouping consecutive push events with the same author+commit.
func (m *Model) buildDisplayItems() {
	m.displayItems = m.displayItems[:0]
	for i := 0; i < len(m.events); {
		e := m.events[i]
		k := event.PushGroupKey(e)

		if k != "" {
			refs := []string{e.PushData.Ref}
			j := i + 1
			for j < len(m.events) && event.PushGroupKey(m.events[j]) == k {
				refs = append(refs, m.events[j].PushData.Ref)
				j++
			}
			if len(refs) > 1 {
				m.displayItems = append(m.displayItems, displayItem{primaryEvent: e, groupedRefs: refs})
				i = j
				continue
			}
		}

		m.displayItems = append(m.displayItems, displayItem{primaryEvent: e})
		i++
	}
}

// SetFilters configures project and group filters. Events whose ProjectName
// doesn't match any filter will be excluded.
func (m *Model) SetFilters(projects, groups []string) {
	m.projectFilters = projects
	m.groupFilters = groups
}

// matchesFilter returns true if the event matches the configured project/group
// filters, or if no filters are set.
func (m Model) matchesFilter(e event.Event) bool {
	if len(m.projectFilters) == 0 && len(m.groupFilters) == 0 {
		return true
	}
	for _, p := range m.projectFilters {
		if strings.Contains(e.ProjectName, p) {
			return true
		}
	}
	for _, g := range m.groupFilters {
		if strings.HasPrefix(e.ProjectName, g+"/") {
			return true
		}
	}
	return false
}

// mergeEvents deduplicates and appends new events to the model's event list,
// maintaining ascending chronological order (oldest first, newest last).
// The API returns events newest-first, so we iterate in reverse to append
// them in chronological order. The list is trimmed from the front (oldest).
func (m *Model) mergeEvents(newEvents []event.Event) {
	for i := len(newEvents) - 1; i >= 0; i-- {
		e := newEvents[i]
		if _, seen := m.seenIDs[e.ID]; seen {
			continue
		}
		if m.clearedAt != nil && e.CreatedAt.Before(*m.clearedAt) {
			continue
		}
		if !m.matchesFilter(e) {
			continue
		}
		m.seenIDs[e.ID] = struct{}{}
		m.events = append(m.events, e)
	}

	if len(m.events) > maxEvents {
		removed := m.events[:len(m.events)-maxEvents]
		for _, e := range removed {
			delete(m.seenIDs, e.ID)
		}
		m.events = m.events[len(m.events)-maxEvents:]
	}
}
