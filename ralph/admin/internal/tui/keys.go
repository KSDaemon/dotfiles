package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Back        key.Binding
	Kill        key.Binding
	Pause       key.Binding
	Refresh     key.Binding
	TogglePanel key.Binding
	Quit        key.Binding
	Yes         key.Binding
	No          key.Binding
}

// DefaultKeyMap returns the default keybindings.
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view details"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Kill: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "kill session"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause/resume"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	TogglePanel: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle panels"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm"),
	),
	No: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "cancel"),
	),
}

// ListHelp returns the help text for the list view.
func ListHelp() string {
	return helpStyle.Render(
		"j/k, up/down: navigate | enter: details | x: kill | p: pause/resume | r: refresh | q: quit",
	)
}

// DetailHelp returns the help text for the detail view.
func DetailHelp(currentPanel panelMode) string {
	nextLabel := currentPanel.next().label()
	return helpStyle.Render(
		"esc: back | tab: show " + nextLabel + " | x: kill | p: pause/resume | q: quit",
	)
}

// ConfirmHelp returns the help text for the confirmation dialog.
func ConfirmHelp() string {
	return helpStyle.Render(
		"y: confirm | n: cancel",
	)
}
