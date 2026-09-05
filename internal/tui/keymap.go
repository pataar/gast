package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all key bindings used by the TUI. Fields are ordered alphabetically for consistency.
type KeyMap struct {
	Clear       key.Binding
	Close       key.Binding
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

// defaultKeyMap returns the default set of key bindings for navigation, refreshing, help, and quitting.
func defaultKeyMap() KeyMap {
	return KeyMap{
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "Clear events"),
		),
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "Close help"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j / down", "Select next event"),
		),
		GoBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G / End", "Select last event"),
		),
		GoTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g / Home", "Select first event"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "Toggle this help"),
		),
		Open: key.NewBinding(
			key.WithKeys("o", "enter"),
			key.WithHelp("o / Enter", "Open event in browser"),
		),
		OpenProject: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "Open project in browser"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q / Ctrl+C", "Quit"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "Force refresh"),
		),
		ToggleTime: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "Toggle relative/absolute time"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k / up", "Select previous event"),
		),
	}
}

// FullHelp returns all key bindings grouped by category, in the order shown in the help overlay.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Down, k.Up, k.GoTop, k.GoBottom},
		{k.Open, k.OpenProject, k.Refresh, k.Clear, k.ToggleTime, k.Help, k.Quit},
	}
}
