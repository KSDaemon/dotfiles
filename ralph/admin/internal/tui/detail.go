package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ksdaemon/ralph-admin/internal/registry"
)

const (
	progressMaxLines = 20
	logMaxBytes      = 32 * 1024 // read last 32KB of log
	logMaxLines      = 40
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
	title := fmt.Sprintf("Ralph Admin - Session Details: %s", sess.Project)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Session info
	b.WriteString(detailSectionStyle.Render("Session Info"))
	b.WriteString("\n")
	b.WriteString(detailRow("PID", fmt.Sprintf("%d", sess.PID)))
	b.WriteString(detailRow("Session ID", sess.SessionID))
	b.WriteString(detailRow("Status", statusStyle(sess.DisplayStatus()).Render(sess.DisplayStatus())))
	b.WriteString("\n")

	// Project info
	b.WriteString(detailSectionStyle.Render("Project"))
	b.WriteString("\n")
	b.WriteString(detailRow("Project", sess.Project))
	b.WriteString(detailRow("Branch", sess.Branch))
	b.WriteString(detailRow("Work Directory", sess.WorkDir))
	if sess.UseWorktree && sess.WorktreeDir != "" {
		b.WriteString(detailRow("Worktree Dir", sess.WorktreeDir))
	}
	b.WriteString("\n")

	// Task info
	b.WriteString(detailSectionStyle.Render("Task"))
	b.WriteString("\n")
	b.WriteString(detailRow("Tool", sess.Tool))
	if sess.PRDDescription != "" {
		b.WriteString(detailRow("PRD Description", sess.PRDDescription))
	}
	b.WriteString(detailRow("Iteration", sess.IterationProgress()))
	b.WriteString("\n")

	// Timing
	b.WriteString(detailSectionStyle.Render("Timing"))
	b.WriteString("\n")
	b.WriteString(detailRow("Started", sess.StartedAt.Local().Format("2006-01-02 15:04:05")))
	b.WriteString(detailRow("Uptime", sess.FormatUptime()))
	b.WriteString(detailRow("Last Heartbeat", sess.FormatHeartbeat()))

	// Average iteration time
	if sess.CurrentIteration > 0 {
		avgDuration := sess.Uptime() / max(1, time.Duration(sess.CurrentIteration))
		b.WriteString(detailRow("Avg Iteration", formatDurationShort(avgDuration)))
	}
	b.WriteString("\n")

	// Live output (tail of session log file)
	b.WriteString(detailSectionStyle.Render("Live Output"))
	b.WriteString("\n")
	logContent, err := registry.ReadLogTail(sess.LogFile, logMaxBytes, logMaxLines)
	if err != nil {
		b.WriteString(progressStyle.Render("Error reading log: " + err.Error()))
	} else {
		b.WriteString(progressStyle.Render(logContent))
	}
	b.WriteString("\n")

	// Progress log (last N lines from progress.txt)
	b.WriteString(detailSectionStyle.Render("Progress Log"))
	b.WriteString("\n")
	progress, err := registry.ReadProgressFile(sess.WorkDir, progressMaxLines)
	if err != nil {
		b.WriteString(progressStyle.Render("Error reading progress: " + err.Error()))
	} else {
		b.WriteString(progressStyle.Render(progress))
	}
	b.WriteString("\n")

	// Status message
	if m.statusMsg != "" {
		b.WriteString("\n")
		if m.statusError {
			b.WriteString(errorStyle.Render("  " + m.statusMsg))
		} else {
			b.WriteString(infoStyle.Render("  " + m.statusMsg))
		}
	}

	// Confirmation or help
	if m.confirming != confirmNone {
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render("  " + m.confirmText))
		b.WriteString("\n")
		b.WriteString(ConfirmHelp())
	} else {
		b.WriteString("\n")
		b.WriteString(DetailHelp())
	}

	return b.String()
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
