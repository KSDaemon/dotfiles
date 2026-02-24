// Package tui implements the Bubble Tea TUI for ralph-admin.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ksdaemon/ralph-admin/internal/registry"
	"github.com/ksdaemon/ralph-admin/internal/session"
)

// View represents which screen is currently active.
type view int

const (
	viewList view = iota
	viewDetail
)

// confirmAction represents a pending confirmation.
type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmKill
)

// panelMode controls what is shown in the bottom section of detail view.
type panelMode int

const (
	panelSplit        panelMode = iota // both log and progress side by side
	panelLogOnly                       // only live output
	panelProgressOnly                  // only progress log
)

func (p panelMode) next() panelMode {
	return (p + 1) % 3
}

func (p panelMode) label() string {
	switch p {
	case panelSplit:
		return "split"
	case panelLogOnly:
		return "log"
	case panelProgressOnly:
		return "progress"
	}
	return ""
}

// tickMsg is sent periodically to refresh session data.
type tickMsg time.Time

// sessionsMsg carries refreshed session data.
type sessionsMsg []*session.Session

// actionResultMsg carries the result of an action (kill, pause, resume).
type actionResultMsg struct {
	message string
	isError bool
}

// Model is the top-level Bubble Tea model.
type Model struct {
	registry *registry.Registry
	sessions []*session.Session

	// View state
	currentView view
	cursor      int
	selected    *session.Session

	// Confirmation state
	confirming  confirmAction
	confirmText string

	// Status message (shown temporarily)
	statusMsg   string
	statusError bool

	// Detail view panel mode
	detailPanel panelMode

	// Terminal dimensions
	width  int
	height int

	// Tick interval for auto-refresh
	tickInterval time.Duration
}

// NewModel creates a new top-level model.
func NewModel(reg *registry.Registry) Model {
	return Model{
		registry:     reg,
		currentView:  viewList,
		tickInterval: 1 * time.Second,
	}
}

// Init starts the tick timer and loads initial data.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshSessions,
		m.tick(),
	)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			m.refreshSessions,
			m.tick(),
		)

	case sessionsMsg:
		m.sessions = msg
		// Clamp cursor
		if m.cursor >= len(m.sessions) {
			m.cursor = max(0, len(m.sessions)-1)
		}
		// Update selected session if in detail view
		if m.currentView == viewDetail && m.selected != nil {
			for _, s := range m.sessions {
				if s.SessionID == m.selected.SessionID {
					m.selected = s
					break
				}
			}
		}
		return m, nil

	case actionResultMsg:
		m.statusMsg = msg.message
		m.statusError = msg.isError
		return m, m.refreshSessions

	case tea.KeyMsg:
		// Handle confirmation first
		if m.confirming != confirmNone {
			return m.handleConfirm(msg)
		}

		switch m.currentView {
		case viewList:
			return m.updateList(msg)
		case viewDetail:
			return m.updateDetail(msg)
		}
	}

	return m, nil
}

// View renders the current screen.
func (m Model) View() string {
	switch m.currentView {
	case viewList:
		return m.viewList()
	case viewDetail:
		return m.viewDetail()
	}
	return ""
}

// tick returns a command that sends a tickMsg after the tick interval.
func (m Model) tick() tea.Cmd {
	return tea.Tick(m.tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshSessions loads sessions from the registry.
func (m Model) refreshSessions() tea.Msg {
	sessions, err := m.registry.List()
	if err != nil {
		return actionResultMsg{
			message: "Error reading sessions: " + err.Error(),
			isError: true,
		}
	}
	return sessionsMsg(sessions)
}

// handleConfirm processes y/n responses.
func (m Model) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, DefaultKeyMap.Yes):
		action := m.confirming
		m.confirming = confirmNone
		m.confirmText = ""

		sess := m.getSelectedSession()
		if sess == nil {
			return m, nil
		}

		switch action {
		case confirmKill:
			return m, m.killSession(sess)
		}

	case keyMatches(msg, DefaultKeyMap.No), keyMatches(msg, DefaultKeyMap.Back):
		m.confirming = confirmNone
		m.confirmText = ""
	}

	return m, nil
}

// getSelectedSession returns the currently focused session.
func (m Model) getSelectedSession() *session.Session {
	if m.currentView == viewDetail && m.selected != nil {
		return m.selected
	}
	if m.cursor >= 0 && m.cursor < len(m.sessions) {
		return m.sessions[m.cursor]
	}
	return nil
}

// killSession returns a command that kills the session.
func (m Model) killSession(sess *session.Session) tea.Cmd {
	return func() tea.Msg {
		err := m.registry.KillSession(sess)
		if err != nil {
			return actionResultMsg{
				message: "Kill failed: " + err.Error(),
				isError: true,
			}
		}
		return actionResultMsg{
			message: "Sent SIGTERM to " + sess.Project + " (PID " + itoa(sess.PID) + ")",
			isError: false,
		}
	}
}

// pauseResumeSession returns a command that toggles pause/resume.
func (m Model) pauseResumeSession(sess *session.Session) tea.Cmd {
	return func() tea.Msg {
		if sess.IsTerminal() {
			return actionResultMsg{
				message: "Cannot pause/resume: session is " + sess.Status,
				isError: true,
			}
		}
		if sess.Status == session.StatusPaused {
			err := m.registry.ResumeSession(sess)
			if err != nil {
				return actionResultMsg{
					message: "Resume failed: " + err.Error(),
					isError: true,
				}
			}
			return actionResultMsg{
				message: "Resumed " + sess.Project + " (PID " + itoa(sess.PID) + ")",
				isError: false,
			}
		}

		err := m.registry.PauseSession(sess)
		if err != nil {
			return actionResultMsg{
				message: "Pause failed: " + err.Error(),
				isError: true,
			}
		}
		return actionResultMsg{
			message: "Paused " + sess.Project + " (PID " + itoa(sess.PID) + ")",
			isError: false,
		}
	}
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	for _, k := range binding.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
