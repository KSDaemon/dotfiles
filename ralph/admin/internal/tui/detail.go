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
	logMaxBytes = 32 * 1024 // read last 32KB of log

	// Lines reserved for title (1) + header columns (~8) + footer (2) + gaps (3)
	reservedLines = 14
)

var (
	// Panel border style
	panelBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDim)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			PaddingLeft(1)
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

	// === Bottom panels ===
	panelHeight := m.height - reservedLines
	if panelHeight < 5 {
		panelHeight = 5
	}

	panelWidth := m.width
	if panelWidth < 40 {
		panelWidth = 80
	}

	switch m.detailPanel {
	case panelSplit:
		// Account for 2 borders (left+right) per panel = 4 chars each, plus 1 char gap
		innerPerPanel := (panelWidth - 9) / 2 // 4+4+1 = 9 chars for borders + gap
		if innerPerPanel < 20 {
			innerPerPanel = 20
		}

		logContent := m.getLogContent(panelHeight - 2) // -2 for border top/bottom
		progContent := m.getProgressContent(panelHeight - 2)

		logPanel := renderBorderedPanel("Live Output", logContent, innerPerPanel, panelHeight)
		progPanel := renderBorderedPanel("Progress Log", progContent, innerPerPanel, panelHeight)

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			logPanel,
			" ",
			progPanel,
		))

	case panelLogOnly:
		innerWidth := panelWidth - 4 // 2+2 for borders
		logContent := m.getLogContent(panelHeight - 2)
		b.WriteString(renderBorderedPanel("Live Output", logContent, innerWidth, panelHeight))

	case panelProgressOnly:
		innerWidth := panelWidth - 4
		progContent := m.getProgressContent(panelHeight - 2)
		b.WriteString(renderBorderedPanel("Progress Log", progContent, innerWidth, panelHeight))
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

// renderBorderedPanel renders a panel with a border and title, fixed to exact height.
func renderBorderedPanel(title, content string, innerWidth, totalHeight int) string {
	// Border takes 2 lines (top + bottom)
	contentHeight := totalHeight - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Truncate content to fit
	lines := strings.Split(content, "\n")
	if len(lines) > contentHeight {
		lines = lines[len(lines)-contentHeight:]
	}
	// Pad with empty lines if content is shorter
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	// Truncate each line to fit width
	for i, line := range lines {
		if len(line) > innerWidth {
			lines[i] = line[:innerWidth]
		}
	}

	body := strings.Join(lines, "\n")

	return panelBorderStyle.
		Width(innerWidth).
		Height(contentHeight).
		Render(panelTitleStyle.Render(title) + "\n" + body)
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
