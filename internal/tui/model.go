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
		// Capture only the events so the command doesn't keep the whole model alive.
		events := m.demoEvents
		return func() tea.Msg {
			return EventsFetchedMsg{Events: events}
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
		m.refreshContent()

	case EventsFetchedMsg:
		suppressNotify := m.shouldSuppressNotifications()
		m.fetching = false
		m.initialFetch = false
		m.manualRefresh = false
		m.lastUpdate = time.Now()
		m.err = nil
		m.consecutiveErrs = 0
		if !suppressNotify {
			m.checkMentions(msg.Events)
		}
		added := m.mergeEvents(msg.Events)
		oldItemCount := len(m.displayItems)
		m.buildDisplayItems()
		if m.initialized {
			// Only auto-scroll to bottom if user was already at the end or this is the first fetch.
			wasAtEnd := m.selectedIdx >= oldItemCount-1
			if wasAtEnd && len(m.displayItems) > 0 {
				m.selectedIdx = len(m.displayItems) - 1
			}
			m.refreshContent()
			if wasAtEnd {
				m.viewport.GotoBottom()
			}
		} else if len(m.displayItems) > 0 {
			m.selectedIdx = len(m.displayItems) - 1
		}
		// Lazily resolve truncated commit titles in the background.
		cmds = append(cmds, m.resolveCommitTitles(added))
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
		// Update the commit title on every event sharing the commit and rebuild display.
		for i := range m.events {
			pd := m.events[i].PushData
			if pd != nil && m.events[i].ProjectID == msg.ProjectID && pd.CommitTo == msg.SHA {
				pd.CommitTitle = msg.Title
			}
		}
		m.buildDisplayItems()
		m.refreshContent()

	case TickMsg:
		m.fetching = true
		cmds = append(cmds, m.spinner.Tick, m.fetchCmd())

	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Quit) || key.Matches(msg, m.keys.Close) {
				m.showHelp = false
				m.refreshContent()
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
			if e, ok := m.selectedEvent(); ok {
				if host := m.gitlabHost(); host != "" {
					_ = browser.OpenEvent(host, e)
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.OpenProject):
			if e, ok := m.selectedEvent(); ok && e.ProjectName != "" {
				if host := m.gitlabHost(); host != "" {
					_ = browser.Open(browser.ProjectURL(host, e.ProjectName))
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
			m.refreshContent()
			if m.initialized {
				m.viewport.GotoTop()
			}
			return m, nil
		case key.Matches(msg, m.keys.ToggleTime):
			event.RelativeTime = !event.RelativeTime
			m.refreshContent()
			return m, nil
		case key.Matches(msg, m.keys.Up):
			m.mentionCount = 0
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.refreshContent()
				m.scrollToSelected()
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			m.mentionCount = 0
			if m.selectedIdx < len(m.displayItems)-1 {
				m.selectedIdx++
				m.refreshContent()
				m.scrollToSelected()
			}
			return m, nil
		case key.Matches(msg, m.keys.GoTop):
			m.selectedIdx = 0
			m.refreshContent()
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.GoBottom):
			if len(m.displayItems) > 0 {
				m.selectedIdx = len(m.displayItems) - 1
			}
			m.refreshContent()
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

	// Forward non-key messages to the viewport (window size, mouse wheel, etc.);
	// key messages are handled above via key bindings for event selection.
	if _, isKey := msg.(tea.KeyMsg); !isKey && m.initialized && !m.showHelp {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
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

	return event.AlignLeftRight(title, right, m.width)
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

	return footerStyle.Render(event.AlignLeftRight(left, eventCount, m.width))
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

	blocks := make([]string, 0, len(m.displayItems))
	for i, item := range m.displayItems {
		var block string
		if len(item.groupedRefs) > 1 {
			block = event.FormatGroupedPush(item.primaryEvent, item.groupedRefs, contentWidth)
		} else {
			block = event.FormatEvent(item.primaryEvent, contentWidth)
		}

		// Prefix the first line with the selection indicator and indent continuation lines to match.
		prefix := "  "
		if i == m.selectedIdx {
			prefix = selectedIndicatorStyle.Render("▸ ")
		}
		blocks = append(blocks, prefix+strings.ReplaceAll(block, "\n", "\n  "))
	}
	sep := dividerStyle.Render(strings.Repeat("┄", m.width))
	return strings.Join(blocks, "\n"+sep+"\n")
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(helpTitleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	for _, group := range m.keys.FullHelp() {
		for _, binding := range group {
			h := binding.Help()
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %-14s %s", h.Key, h.Desc)))
			b.WriteString("\n")
		}
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

/*
resolveCommitTitles returns commands to fetch full titles for push events with truncated commit
titles (ending in "..."), deduplicated per commit. Called with the genuinely new events of a fetch
so already-known events don't re-resolve on every poll.
*/
func (m Model) resolveCommitTitles(newEvents []event.Event) tea.Cmd {
	if m.client == nil {
		return nil
	}
	var cmds []tea.Cmd
	requested := make(map[string]struct{})
	for _, e := range newEvents {
		if e.PushData == nil || e.PushData.CommitTo == "" {
			continue
		}
		if !strings.HasSuffix(e.PushData.CommitTitle, "...") {
			continue
		}
		key := fmt.Sprintf("%d:%s", e.ProjectID, e.PushData.CommitTo)
		if _, ok := requested[key]; ok {
			continue
		}
		requested[key] = struct{}{}
		cmds = append(cmds, resolveCommitTitleCmd(m.client, e.ProjectID, e.PushData.CommitTo, e.PushData.CommitTitle))
	}
	return tea.Batch(cmds...)
}

// refreshContent re-renders the event list into the viewport once it exists.
func (m *Model) refreshContent() {
	if m.initialized {
		m.viewport.SetContent(m.renderEvents())
	}
}

// selectedEvent returns the primary event of the selected display item.
func (m Model) selectedEvent() (event.Event, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.displayItems) {
		return event.Event{}, false
	}
	return m.displayItems[m.selectedIdx].primaryEvent, true
}

// itemLineCount returns the number of rendered lines for a display item.
func (m Model) itemLineCount(idx int) int {
	if idx < 0 || idx >= len(m.displayItems) {
		return 1
	}
	if event.HasDetailLine(m.displayItems[idx].primaryEvent) {
		return 2
	}
	return 1
}

// gitlabHost returns the configured GitLab host URL.
func (m Model) gitlabHost() string {
	return m.cfg.GitLabHost
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

// shouldSuppressNotifications returns true when notifications should be skipped:
// during the initial fetch or the first fetch after clearing events.
func (m Model) shouldSuppressNotifications() bool {
	return m.initialFetch || (m.clearedAt != nil && len(m.events) == 0)
}

// checkMentions scans new events for @mentions of the current user. When found,
// increments the mention counter and optionally sends a desktop notification.
func (m *Model) checkMentions(newEvents []event.Event) {
	if event.CurrentUser == "" {
		return
	}
	for _, e := range newEvents {
		if _, seen := m.seenIDs[e.ID]; seen {
			continue
		}
		if e.AuthorUsername == event.CurrentUser {
			continue
		}
		if !event.HasMention(e.NoteBody) {
			continue
		}
		m.mentionCount++
		if m.cfg.Notifications {
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
		k, groupable := event.PushGroupKey(e)

		if groupable {
			refs := []string{e.PushData.Ref}
			j := i + 1
			for j < len(m.events) {
				jk, ok := event.PushGroupKey(m.events[j])
				if !ok || jk != k {
					break
				}
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

/*
mergeEvents deduplicates and appends new events to the model's event list, maintaining ascending
chronological order (oldest first, newest last). The API returns events newest-first, so we iterate
in reverse to append them in chronological order. The list is trimmed from the front (oldest).
Returns the events that were actually added.
*/
func (m *Model) mergeEvents(newEvents []event.Event) []event.Event {
	var added []event.Event
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
		added = append(added, e)
	}

	if len(m.events) > maxEvents {
		removed := m.events[:len(m.events)-maxEvents]
		for _, e := range removed {
			delete(m.seenIDs, e.ID)
		}
		m.events = m.events[len(m.events)-maxEvents:]
	}

	return added
}
