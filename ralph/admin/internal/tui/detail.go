package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ksdaemon/ralph-admin/internal/registry"
)

const (
	progressMaxLines = 20
	logMaxBytes      = 32 * 1024 // read last 32KB of log
	logMaxLines      = 50

	// How many lines the header section (info + timing) takes roughly
	headerReservedLines = 12
	// Footer (help + status) reserved lines
	footerReservedLines = 4
)

// updateDetail handles key events in the detail view.
func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, DefaultKeyMap.Back):
		m.currentView = viewList
		m.selected = nil
		m.statusMsg = ""
		m.confirming = confirmNone
		m.confirmText = ""

	case keyMatches(msg, DefaultKeyMap.Quit):
		return m, tea.Quit

	case keyMatches(msg, DefaultKeyMap.Kill):
		if m.selected != nil {
			m.confirming = confirmKill
			m.confirmText = fmt.Sprintf("Kill session %s (PID %d)? [y/n]", m.selected.Project, m.selected.PID)
		}

	case keyMatches(msg, DefaultKeyMap.Pause):
		if m.selected != nil {
			return m, m.pauseResumeSession(m.selected)
		}

	case keyMatches(msg, DefaultKeyMap.TogglePanel):
		m.detailPanel = m.detailPanel.next()
	}

	return m, nil
}

// viewDetail renders the detail screen.
func (m Model) viewDetail() string {
	if m.selected == nil {
		m.currentView = viewList
		return m.viewList()
	}

	sess := m.selected
	var b strings.Builder

	// Title
	title := fmt.Sprintf("Ralph Admin - %s", sess.Project)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// === Two-column header: Info (left) | Timing (right) ===
	halfWidth := m.width / 2
	if halfWidth < 40 {
		halfWidth = 40
	}

	// Left column: session & project info
	var left strings.Builder
	left.WriteString(detailRow("Status", statusStyle(sess.DisplayStatus()).Render(sess.DisplayStatus())))
	left.WriteString(detailRow("Branch", sess.Branch))
	left.WriteString(detailRow("Directory", sess.WorkDir))
	if sess.UseWorktree && sess.WorktreeDir != "" {
		left.WriteString(detailRow("Worktree", sess.WorktreeDir))
	}
	left.WriteString(detailRow("Tool", sess.Tool))
	if sess.PRDDescription != "" {
		desc := sess.PRDDescription
		maxDesc := halfWidth - 22
		if maxDesc > 0 && len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}
		left.WriteString(detailRow("PRD", desc))
	}

	// Right column: timing & iteration
	var right strings.Builder
	right.WriteString(detailRow("PID", fmt.Sprintf("%d", sess.PID)))
	right.WriteString(detailRow("Iteration", sess.IterationProgress()))
	right.WriteString(detailRow("Started", sess.StartedAt.Local().Format("2006-01-02 15:04:05")))
	right.WriteString(detailRow("Uptime", sess.FormatUptime()))
	right.WriteString(detailRow("Heartbeat", sess.FormatHeartbeat()))
	if sess.CurrentIteration > 0 {
		avgDuration := sess.Uptime() / max(1, time.Duration(sess.CurrentIteration))
		right.WriteString(detailRow("Avg Iter", formatDurationShort(avgDuration)))
	}

	leftBlock := lipgloss.NewStyle().Width(halfWidth).Render(left.String())
	rightBlock := lipgloss.NewStyle().Width(halfWidth).Render(right.String())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightBlock))
	b.WriteString("\n")

	// === Bottom panels: log / progress / split ===
	panelHeight := m.height - headerReservedLines - footerReservedLines
	if panelHeight < 5 {
		panelHeight = 5
	}

	panelWidth := m.width
	if panelWidth < 40 {
		panelWidth = 80
	}

	switch m.detailPanel {
	case panelSplit:
		splitWidth := (panelWidth - 3) / 2 // 3 for " | " separator
		logContent := m.getLogContent(panelHeight)
		progContent := m.getProgressContent(panelHeight)

		logPanel := renderPanel("Live Output", logContent, splitWidth, panelHeight)
		progPanel := renderPanel("Progress Log", progContent, splitWidth, panelHeight)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			logPanel,
			lipgloss.NewStyle().Foreground(colorDim).Render(" | "),
			progPanel,
		))

	case panelLogOnly:
		logContent := m.getLogContent(panelHeight)
		b.WriteString(renderPanel("Live Output", logContent, panelWidth, panelHeight))

	case panelProgressOnly:
		progContent := m.getProgressContent(panelHeight)
		b.WriteString(renderPanel("Progress Log", progContent, panelWidth, panelHeight))
	}

	b.WriteString("\n")

	// Status message
	if m.statusMsg != "" {
		if m.statusError {
			b.WriteString(errorStyle.Render("  " + m.statusMsg))
		} else {
			b.WriteString(infoStyle.Render("  " + m.statusMsg))
		}
		b.WriteString("\n")
	}

	// Confirmation or help
	if m.confirming != confirmNone {
		b.WriteString(confirmStyle.Render("  " + m.confirmText))
		b.WriteString("\n")
		b.WriteString(ConfirmHelp())
	} else {
		b.WriteString(DetailHelp(m.detailPanel))
	}

	return b.String()
}

// getLogContent reads the session log tail.
func (m Model) getLogContent(maxLines int) string {
	if m.selected == nil {
		return ""
	}
	content, err := registry.ReadLogTail(m.selected.LogFile, logMaxBytes, maxLines)
	if err != nil {
		return "Error: " + err.Error()
	}
	return content
}

// getProgressContent reads the progress.txt tail.
func (m Model) getProgressContent(maxLines int) string {
	if m.selected == nil {
		return ""
	}
	content, err := registry.ReadProgressFile(m.selected.WorkDir, maxLines)
	if err != nil {
		return "Error: " + err.Error()
	}
	return content
}

// renderPanel renders a titled panel with content, constrained to width and height.
func renderPanel(title, content string, width, height int) string {
	titleStr := detailSectionStyle.Render(title)

	// Constrain content to height lines
	lines := strings.Split(content, "\n")
	if len(lines) > height-2 { // -2 for title + padding
		lines = lines[len(lines)-(height-2):]
	}

	contentStr := progressStyle.
		Width(width - 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, titleStr, contentStr)
}

func detailRow(label, value string) string {
	return fmt.Sprintf("  %s %s\n",
		detailLabelStyle.Render(label+":"),
		detailValueStyle.Render(value),
	)
}

func formatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
