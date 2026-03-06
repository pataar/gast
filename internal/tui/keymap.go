package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all key bindings used by the TUI. Fields are ordered
// alphabetically for consistency.
type KeyMap struct {
	Down        key.Binding
	GoBottom    key.Binding
	GoTop       key.Binding
	Help        key.Binding
	Open        key.Binding
	OpenProject key.Binding
	Quit        key.Binding
	Refresh     key.Binding
	ToggleTime  key.Binding
	Up          key.Binding
}

// defaultKeyMap returns the default set of key bindings for navigation,
// refreshing, help, and quitting.
func defaultKeyMap() KeyMap {
	return KeyMap{
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "scroll down"),
		),
		GoBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "go to bottom"),
		),
		GoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "go to top"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Open: key.NewBinding(
			key.WithKeys("o", "enter"),
			key.WithHelp("o/enter", "open in browser"),
		),
		OpenProject: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "open project in browser"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		ToggleTime: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle relative time"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "scroll up"),
		),
	}
}

// ShortHelp returns a compact set of key bindings for the help bubble.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Up, k.Down, k.Refresh, k.Help}
}

// FullHelp returns the complete set of key bindings grouped by category for
// the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.GoTop, k.GoBottom},
		{k.Open, k.OpenProject, k.Refresh, k.ToggleTime, k.Help, k.Quit},
	}
}
