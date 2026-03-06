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
	"github.com/pataar/gast/internal/config"
	"github.com/pataar/gast/internal/event"
	"github.com/pataar/gast/internal/gitlab"
)

// maxEvents is the upper bound on events kept in memory. Older events beyond
// this limit are discarded to prevent unbounded memory growth.
const maxEvents = 500

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

	fetching    bool
	lastUpdate  time.Time
	err         error
	showHelp    bool
	initialized bool
	demo        bool
	demoEvents  []event.Event
}

// NewModel creates a new TUI model wired to the given GitLab client and config.
func NewModel(client *gitlab.Client, cfg *config.Config) Model {
	s := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(spinnerStyle))

	// Set the current username so @mentions can be highlighted in note bodies.
	event.CurrentUser = cfg.Username

	return Model{
		client:  client,
		cfg:     cfg,
		seenIDs: make(map[int]struct{}),
		spinner: s,
		keys:    defaultKeyMap(),
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
		m.lastUpdate = time.Now()
		m.err = nil
		m.mergeEvents(msg.Events)
		if m.initialized {
			m.viewport.SetContent(m.renderEvents())
			m.viewport.GotoBottom()
		}
		// Schedule the next poll after a successful fetch (skip in demo mode).
		if !m.demo {
			cmds = append(cmds, tickCmd(m.cfg.PollInterval))
		}

	case FetchErrorMsg:
		m.fetching = false
		m.err = msg.Err
		// Retry after the normal interval even on error.
		cmds = append(cmds, tickCmd(m.cfg.PollInterval))

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
		case key.Matches(msg, m.keys.Refresh):
			m.fetching = true
			return m, tea.Batch(m.spinner.Tick, m.fetchCmd())
		case key.Matches(msg, m.keys.GoTop):
			m.viewport.GotoTop()
			return m, nil
		case key.Matches(msg, m.keys.GoBottom):
			m.viewport.GotoBottom()
			return m, nil
		}

	case spinner.TickMsg:
		// Only animate the spinner while a fetch is in progress to avoid
		// unnecessary re-renders (which cause visible flashing).
		if m.fetching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.initialized && !m.showHelp {
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

	right := ""
	if m.fetching {
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
	left := " q quit  j/k scroll  r refresh  ? help"

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
	if len(m.events) == 0 {
		return "\n  No events yet. Waiting for first fetch..."
	}

	var blocks []string
	for i := 0; i < len(m.events); {
		e := m.events[i]
		key := event.PushGroupKey(e)

		// Group consecutive push events with the same author + commit title.
		if key != "" {
			refs := []string{e.PushData.Ref}
			j := i + 1
			for j < len(m.events) && event.PushGroupKey(m.events[j]) == key {
				refs = append(refs, m.events[j].PushData.Ref)
				j++
			}
			if len(refs) > 1 {
				blocks = append(blocks, event.FormatGroupedPush(e, refs, m.width))
				i = j
				continue
			}
		}

		blocks = append(blocks, event.FormatEvent(e, m.width))
		i++
	}
	// Separate events with a thin dimmed dotted line for visual clarity
	// without taking too much vertical space.
	sep := dividerStyle.Render(strings.Repeat("┄", m.width))
	return strings.Join(blocks, "\n"+sep+"\n")
}

func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(helpTitleStyle.Render("Keybindings"))
	b.WriteString("\n\n")

	bindings := []struct{ key, desc string }{
		{"j / down", "Scroll down"},
		{"k / up", "Scroll up"},
		{"g / Home", "Go to top"},
		{"G / End", "Go to bottom"},
		{"r", "Force refresh"},
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

// fetchCmd builds the fetch command using the latest event's timestamp as the
// "after" filter, or nil for the initial fetch.
func (m Model) fetchCmd() tea.Cmd {
	var after *time.Time
	if len(m.events) > 0 {
		t := m.events[len(m.events)-1].CreatedAt
		after = &t
	}
	return fetchEventsCmd(m.client, after, m.cfg.PageSize)
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
