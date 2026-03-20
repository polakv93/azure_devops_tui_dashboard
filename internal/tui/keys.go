package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines the keybindings for the application
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	Enter    key.Binding
	Space    key.Binding
	Back     key.Binding
	Logs     key.Binding
	Notify   key.Binding
	Create   key.Binding
	Run      key.Binding
	Refresh  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("PgUp/b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f"),
			key.WithHelp("PgDn/f", "page down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev project"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next project"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch section"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open in browser"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle option"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel/back"),
		),
		Logs: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "view logs"),
		),
		Notify: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "toggle notify"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create PR"),
		),
		Run: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "run pipeline"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Up, k.Down, k.Logs, k.Enter, k.Create, k.Run, k.Notify, k.Refresh, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Left, k.Right, k.Tab},
		{k.Tab, k.Enter, k.Space, k.Back},
		{k.Logs, k.Create, k.Run, k.Notify, k.Refresh},
		{k.Help, k.Quit},
	}
}
