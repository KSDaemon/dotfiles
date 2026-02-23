package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Minimum column widths for fixed-width columns
const (
	minColTool      = 10
	minColIteration = 10
	minColStatus    = 12
	minColHeartbeat = 12
	minColUptime    = 8
	minColProject   = 10
	minColBranch    = 12
	tablePadding    = 4 // 2 chars left margin + gaps between columns
)

// columnWidths holds the computed column widths for the current terminal size.
type columnWidths struct {
	project   int
	branch    int
	tool      int
	iteration int
	status    int
	heartbeat int
	uptime    int
}

// computeColumns calculates column widths based on available terminal width.
func computeColumns(termWidth int) columnWidths {
	if termWidth <= 0 {
		termWidth = 120 // reasonable default
	}

	cw := columnWidths{
		tool:      minColTool,
		iteration: minColIteration,
		status:    minColStatus,
		heartbeat: minColHeartbeat,
		uptime:    minColUptime,
	}

	// Fixed columns total + padding (2 left margin + 6 column gaps of 1 space)
	fixedTotal := cw.tool + cw.iteration + cw.status + cw.heartbeat + cw.uptime
	overhead := tablePadding + 5 // 5 gaps between 6 remaining fixed cols
	remaining := termWidth - fixedTotal - overhead

	if remaining < minColProject+minColBranch+2 {
		// Terminal is very narrow — give minimums
		cw.project = minColProject
		cw.branch = minColBranch
	} else {
		// Split remaining between project (35%) and branch (65%)
		cw.project = remaining * 35 / 100
		cw.branch = remaining - cw.project
		if cw.project < minColProject {
			cw.project = minColProject
			cw.branch = remaining - cw.project
		}
		if cw.branch < minColBranch {
			cw.branch = minColBranch
			cw.project = remaining - cw.branch
		}
	}

	return cw
}

// updateList handles key events in the list view.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, DefaultKeyMap.Quit):
		return m, tea.Quit

	case keyMatches(msg, DefaultKeyMap.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case keyMatches(msg, DefaultKeyMap.Down):
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
		}

	case keyMatches(msg, DefaultKeyMap.Enter):
		if len(m.sessions) > 0 && m.cursor < len(m.sessions) {
			m.selected = m.sessions[m.cursor]
			m.currentView = viewDetail
			m.statusMsg = ""
		}

	case keyMatches(msg, DefaultKeyMap.Kill):
		sess := m.getSelectedSession()
		if sess != nil {
			m.confirming = confirmKill
			m.confirmText = fmt.Sprintf("Kill session %s (PID %d)? [y/n]", sess.Project, sess.PID)
		}

	case keyMatches(msg, DefaultKeyMap.Pause):
		sess := m.getSelectedSession()
		if sess != nil {
			return m, m.pauseResumeSession(sess)
		}

	case keyMatches(msg, DefaultKeyMap.Refresh):
		m.statusMsg = ""
		return m, m.refreshSessions
	}

	return m, nil
}

// viewList renders the list screen.
func (m Model) viewList() string {
	var b strings.Builder

	cw := computeColumns(m.width)

	// Title with summary
	activeCount := 0
	pausedCount := 0
	finishedCount := 0
	for _, s := range m.sessions {
		switch s.DisplayStatus() {
		case "running", "stale":
			activeCount++
		case "paused":
			pausedCount++
		case "completed", "interrupted", "max_iterations_reached", "dead":
			finishedCount++
		}
	}

	title := fmt.Sprintf("Ralph Admin - %d session(s)", len(m.sessions))
	var parts []string
	if activeCount > 0 {
		parts = append(parts, fmt.Sprintf("%d active", activeCount))
	}
	if pausedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d paused", pausedCount))
	}
	if finishedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d finished", finishedCount))
	}
	if len(parts) > 0 {
		title += " (" + strings.Join(parts, ", ") + ")"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(m.sessions) == 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("  No ralph sessions found (recent 24h)."))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render("  Start a ralph loop in any project and it will appear here."))
		b.WriteString("\n")
	} else {
		// Table header
		header := renderRow(cw, "PROJECT", "BRANCH", "TOOL", "ITERATION", "STATUS", "HEARTBEAT", "UPTIME", lipgloss.Style{})
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")

		// Table rows
		for i, sess := range m.sessions {
			status := sess.DisplayStatus()

			project := truncate(sess.Project, cw.project)
			branch := truncate(sess.Branch, cw.branch)
			tool := truncate(sess.Tool, cw.tool)
			iter := pad(sess.IterationProgress(), cw.iteration)
			stText := pad(status, cw.status)
			hb := pad(sess.FormatHeartbeat(), cw.heartbeat)
			uptime := pad(sess.FormatUptime(), cw.uptime)

			// Build the row: plain parts + colored status
			row := fmt.Sprintf("  %-*s %-*s %-*s %s %s %s %s",
				cw.project, project,
				cw.branch, branch,
				cw.tool, tool,
				iter,
				statusStyle(status).Render(stText),
				hb,
				uptime,
			)

			if i == m.cursor {
				b.WriteString(selectedStyle.Width(m.width).Render(row))
			} else {
				b.WriteString(row)
			}
			b.WriteString("\n")
		}
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString("\n")
		if m.statusError {
			b.WriteString(errorStyle.Render("  " + m.statusMsg))
		} else {
			b.WriteString(infoStyle.Render("  " + m.statusMsg))
		}
		b.WriteString("\n")
	}

	// Confirmation dialog
	if m.confirming != confirmNone {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render("  " + m.confirmText))
		b.WriteString("\n")
		b.WriteString(ConfirmHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(ListHelp())
	}

	return b.String()
}

// renderRow builds a plain-text table row with the given column widths.
func renderRow(cw columnWidths, project, branch, tool, iteration, status, heartbeat, uptime string, _ lipgloss.Style) string {
	return fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		cw.project, project,
		cw.branch, branch,
		cw.tool, tool,
		cw.iteration, iteration,
		cw.status, status,
		cw.heartbeat, heartbeat,
		cw.uptime, uptime,
	)
}

// truncate shortens a string to maxLen, adding "..." if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// pad right-pads a string to exactly width characters.
func pad(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
