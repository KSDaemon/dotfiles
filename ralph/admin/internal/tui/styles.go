package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorGreen   = lipgloss.Color("#00FF00")
	colorYellow  = lipgloss.Color("#FFFF00")
	colorRed     = lipgloss.Color("#FF0000")
	colorCyan    = lipgloss.Color("#00FFFF")
	colorMagenta = lipgloss.Color("#FF00FF")
	colorGray    = lipgloss.Color("#808080")
	colorWhite   = lipgloss.Color("#FFFFFF")
	colorDim     = lipgloss.Color("#666666")

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			MarginBottom(1)

	// Header row in table
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Underline(true)

	// Selected row
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("#333366"))

	// Status colors
	statusRunningStyle     = lipgloss.NewStyle().Foreground(colorGreen)
	statusCompletedStyle   = lipgloss.NewStyle().Foreground(colorCyan)
	statusInterruptedStyle = lipgloss.NewStyle().Foreground(colorMagenta)
	statusMaxIterStyle     = lipgloss.NewStyle().Foreground(colorYellow)
	statusPausedStyle      = lipgloss.NewStyle().Foreground(colorYellow)
	statusStaleStyle       = lipgloss.NewStyle().Foreground(colorYellow).Bold(true)
	statusDeadStyle        = lipgloss.NewStyle().Foreground(colorRed)

	// Help bar at the bottom
	helpStyle = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginTop(1)

	// Detail view styles
	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorCyan).
				Width(20)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorWhite)

	// Error/info messages
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	// Confirmation dialog
	confirmStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorYellow).
			MarginTop(1)
)

// statusColor returns just the foreground color for a session status.
func statusColor(status string) lipgloss.Color {
	switch status {
	case "running":
		return colorGreen
	case "completed":
		return colorCyan
	case "interrupted":
		return colorMagenta
	case "max_iterations_reached":
		return colorYellow
	case "paused":
		return colorYellow
	case "stale":
		return colorYellow
	case "dead":
		return colorRed
	default:
		return colorGray
	}
}

// statusStyle returns the appropriate style for a session status.
func statusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return statusRunningStyle
	case "completed":
		return statusCompletedStyle
	case "interrupted":
		return statusInterruptedStyle
	case "max_iterations_reached":
		return statusMaxIterStyle
	case "paused":
		return statusPausedStyle
	case "stale":
		return statusStaleStyle
	case "dead":
		return statusDeadStyle
	default:
		return lipgloss.NewStyle().Foreground(colorGray)
	}
}
