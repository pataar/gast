package tui

import "charm.land/lipgloss/v2"

// Lipgloss styles for the TUI layout components.
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")).
			PaddingLeft(1)

	footerStyle = lipgloss.NewStyle().
			Faint(true).
			PaddingLeft(1)

	dividerStyle = lipgloss.NewStyle().
			Faint(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			PaddingLeft(1)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5"))

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5")).
			PaddingLeft(1).
			PaddingBottom(1)

	helpStyle = lipgloss.NewStyle().
			PaddingLeft(2)
)
