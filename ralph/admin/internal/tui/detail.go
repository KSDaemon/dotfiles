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
)

var (
	// Border style for all panels
	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim)
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
	termW := m.width
	if termW < 60 {
		termW = 80
	}

	// === Title (1 line + newline) ===
	title := titleStyle.Render(fmt.Sprintf("Ralph Admin - %s", sess.Project))

	// === Header: two bordered boxes side by side ===
	// Each box gets roughly half the width minus gap and borders
	// Border takes 2 chars on each side = 4 per box, plus 1 gap = 9 total
	headerInnerW := (termW - 9) / 2
	if headerInnerW < 30 {
		headerInnerW = 30
	}

	// Left box: project info
	var leftRows []string
	leftRows = append(leftRows, detailRowStr("Status", statusStyle(sess.DisplayStatus()).Render(sess.DisplayStatus())))
	leftRows = append(leftRows, detailRowStr("Branch", sess.Branch))
	leftRows = append(leftRows, detailRowStr("Directory", sess.WorkDir))
	if sess.UseWorktree && sess.WorktreeDir != "" {
		leftRows = append(leftRows, detailRowStr("Worktree", sess.WorktreeDir))
	}
	leftRows = append(leftRows, detailRowStr("Tool", sess.Tool))
	if sess.PRDDescription != "" {
		desc := sess.PRDDescription
		maxDesc := headerInnerW - 14 // 14 = label width + spacing
		if maxDesc > 10 && len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}
		leftRows = append(leftRows, detailRowStr("PRD", desc))
	}

	// Right box: timing
	var rightRows []string
	rightRows = append(rightRows, detailRowStr("PID", fmt.Sprintf("%d", sess.PID)))
	rightRows = append(rightRows, detailRowStr("Iteration", sess.IterationProgress()))
	rightRows = append(rightRows, detailRowStr("Started", sess.StartedAt.Local().Format("2006-01-02 15:04:05")))
	rightRows = append(rightRows, detailRowStr("Uptime", sess.FormatUptime()))
	rightRows = append(rightRows, detailRowStr("Heartbeat", sess.FormatHeartbeat()))
	if sess.CurrentIteration > 0 {
		avgDuration := sess.Uptime() / max(1, time.Duration(sess.CurrentIteration))
		rightRows = append(rightRows, detailRowStr("Avg Iter", formatDurationShort(avgDuration)))
	}

	// Equalize row count so both boxes have same height
	maxRows := len(leftRows)
	if len(rightRows) > maxRows {
		maxRows = len(rightRows)
	}
	for len(leftRows) < maxRows {
		leftRows = append(leftRows, "")
	}
	for len(rightRows) < maxRows {
		rightRows = append(rightRows, "")
	}

	leftBox := borderStyle.Width(headerInnerW).Render(strings.Join(leftRows, "\n"))
	rightBox := borderStyle.Width(headerInnerW).Render(strings.Join(rightRows, "\n"))
	header := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)

	// Count how many lines the header section takes
	headerLines := 1 + lipgloss.Height(header) // 1 for title
	// Footer: 1 line for help (+ 1 potential status message)
	footerLines := 2

	// === Bottom panels ===
	// Total height for panels = terminal height - header - footer
	panelTotalH := m.height - headerLines - footerLines
	if panelTotalH < 5 {
		panelTotalH = 5
	}
	// Inner height = total - 2 (border top + bottom) - 1 (title inside panel)
	panelContentH := panelTotalH - 3
	if panelContentH < 2 {
		panelContentH = 2
	}

	var panels string
	switch m.detailPanel {
	case panelSplit:
		panelInnerW := (termW - 9) / 2 // same math as header
		if panelInnerW < 20 {
			panelInnerW = 20
		}
		logContent := m.getLogContent(panelContentH)
		progContent := m.getProgressContent(panelContentH)

		logPanel := renderPanel("Live Output", logContent, panelInnerW, panelContentH)
		progPanel := renderPanel("Progress Log", progContent, panelInnerW, panelContentH)
		panels = lipgloss.JoinHorizontal(lipgloss.Top, logPanel, " ", progPanel)

	case panelLogOnly:
		panelInnerW := termW - 4 // 2 border chars each side
		logContent := m.getLogContent(panelContentH)
		panels = renderPanel("Live Output", logContent, panelInnerW, panelContentH)

	case panelProgressOnly:
		panelInnerW := termW - 4
		progContent := m.getProgressContent(panelContentH)
		panels = renderPanel("Progress Log", progContent, panelInnerW, panelContentH)
	}

	// === Footer ===
	var footer string
	if m.statusMsg != "" {
		if m.statusError {
			footer = errorStyle.Render("  "+m.statusMsg) + "\n"
		} else {
			footer = infoStyle.Render("  "+m.statusMsg) + "\n"
		}
	}
	if m.confirming != confirmNone {
		footer += confirmStyle.Render("  "+m.confirmText) + "\n" + ConfirmHelp()
	} else {
		footer += DetailHelp(m.detailPanel)
	}

	// === Assemble everything ===
	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		header,
		panels,
		footer,
	)
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

// renderPanel creates a bordered panel with a title header and fixed-height content.
// The title is rendered above the content, inside the border.
// innerWidth is the content width (border adds 2 chars each side).
// contentLines is how many lines of text content to show.
func renderPanel(title, content string, innerWidth, contentLines int) string {
	// Prepare content: take last N lines, pad if shorter
	lines := strings.Split(content, "\n")
	if len(lines) > contentLines {
		lines = lines[len(lines)-contentLines:]
	}
	for len(lines) < contentLines {
		lines = append(lines, "")
	}
	// Truncate each line to inner width
	for i, line := range lines {
		if len(line) > innerWidth-1 {
			lines[i] = line[:innerWidth-1]
		}
	}

	titleStr := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorCyan).
		Render(title)

	body := titleStr + "\n" + strings.Join(lines, "\n")

	return borderStyle.
		Width(innerWidth).
		Render(body)
}

func detailRowStr(label, value string) string {
	return fmt.Sprintf(" %s %s",
		detailLabelStyle.Render(label+":"),
		detailValueStyle.Render(value),
	)
}

func detailRow(label, value string) string {
	return detailRowStr(label, value) + "\n"
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
