package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors
var (
	ColorGreen   = lipgloss.Color("#00FF00")
	ColorRed     = lipgloss.Color("#FF0000")
	ColorYellow  = lipgloss.Color("#FFFF00")
	ColorOrange  = lipgloss.Color("#FFA500")
	ColorGray    = lipgloss.Color("#808080")
	ColorWhite   = lipgloss.Color("#FFFFFF")
	ColorBlue    = lipgloss.Color("#00BFFF")
	ColorPurple  = lipgloss.Color("#9370DB")
	ColorCyan    = lipgloss.Color("#00CED1")
	ColorDimGray = lipgloss.Color("#696969")
)

// Status styles
var (
	SucceededStyle  = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	FailedStyle     = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	InProgressStyle = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	CanceledStyle   = lipgloss.NewStyle().Foreground(ColorOrange)
	QueuedStyle     = lipgloss.NewStyle().Foreground(ColorGray)
	NotStartedStyle = lipgloss.NewStyle().Foreground(ColorGray)
)

// UI element styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorPurple)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorDimGray)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorWhite).
			Background(ColorBlue).
			Bold(true)

	ProjectTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(ColorDimGray)

	ActiveProjectStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(ColorWhite).
				Bold(true).
				Underline(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorDimGray).
			MarginTop(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorDimGray)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWhite).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorDimGray)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3A3A3A")).
				Foreground(ColorWhite)
)

// GetStatusStyle returns the appropriate style for a build/release status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "succeeded":
		return SucceededStyle
	case "failed", "rejected":
		return FailedStyle
	case "inProgress":
		return InProgressStyle
	case "canceled":
		return CanceledStyle
	case "queued", "scheduled":
		return QueuedStyle
	case "notStarted":
		return NotStartedStyle
	case "partiallySucceeded":
		return lipgloss.NewStyle().Foreground(ColorOrange)
	default:
		return lipgloss.NewStyle().Foreground(ColorGray)
	}
}

// FormatStatus formats a status string with appropriate styling
func FormatStatus(status string) string {
	style := GetStatusStyle(status)
	return style.Render(status)
}

// Box styles for layout
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDimGray).
			Padding(1, 2)

	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBlue).
			Padding(1, 2)
)
