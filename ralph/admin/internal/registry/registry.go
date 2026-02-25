// Package registry handles reading, validating, and managing ralph session files.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/ksdaemon/ralph-admin/internal/session"
)

// Registry manages ralph session files in the registry directory.
type Registry struct {
	Dir string
}

// New creates a new Registry pointing to the default registry directory.
// It uses $TMPDIR/ralph-sessions/ (which on macOS is user-scoped).
func New() *Registry {
	tmpDir := os.TempDir()
	dir := filepath.Join(tmpDir, "ralph-sessions")
	return &Registry{Dir: dir}
}

// NewWithDir creates a Registry with a custom directory (useful for testing).
func NewWithDir(dir string) *Registry {
	return &Registry{Dir: dir}
}

// List reads all session files from the registry and enriches them
// with liveness info. Expired sessions (>24h) are cleaned up.
// Duplicate sessions for the same work_dir are deduplicated: only the
// newest session per directory is kept, older ones are removed.
func (r *Registry) List() ([]*session.Session, error) {
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no registry dir = no sessions
		}
		return nil, fmt.Errorf("reading registry dir: %w", err)
	}

	var all []*session.Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(r.Dir, entry.Name())
		sess, err := r.readSession(filePath)
		if err != nil {
			// Corrupted file — remove it
			os.Remove(filePath)
			continue
		}

		// Extract session ID from filename (remove .json suffix)
		sess.SessionID = strings.TrimSuffix(entry.Name(), ".json")
		sess.FilePath = filePath

		// Remove expired sessions (older than 24h)
		if sess.IsExpired() {
			r.removeSessionFiles(sess)
			continue
		}

		// Check if process is alive
		sess.IsAlive = isProcessAlive(sess.PID)

		all = append(all, sess)
	}

	// Deduplicate: keep only the newest session per real work directory.
	// Resolve symlinks so worktree paths pointing to the same dir are deduped.
	best := make(map[string]*session.Session)
	for _, sess := range all {
		key := resolveRealPath(sess.WorkDir)
		existing, found := best[key]
		if !found {
			best[key] = sess
			continue
		}
		// Prefer running sessions over terminal ones
		// If same status class, prefer newer
		if r.sessionPriority(sess) > r.sessionPriority(existing) {
			r.removeSessionFiles(existing)
			best[key] = sess
		} else if r.sessionPriority(sess) == r.sessionPriority(existing) && sess.StartedAt.After(existing.StartedAt) {
			r.removeSessionFiles(existing)
			best[key] = sess
		} else {
			r.removeSessionFiles(sess)
		}
	}

	var sessions []*session.Session
	for _, sess := range best {
		sessions = append(sessions, sess)
	}

	// Stable sort: active first, then by project name, then newest first
	sort.Slice(sessions, func(i, j int) bool {
		a, b := sessions[i], sessions[j]
		aPri := r.sessionPriority(a)
		bPri := r.sessionPriority(b)
		if aPri != bPri {
			return aPri > bPri // active sessions first
		}
		if a.Project != b.Project {
			return a.Project < b.Project // alphabetical
		}
		return a.StartedAt.After(b.StartedAt) // newest first
	})

	return sessions, nil
}

// sessionPriority returns a priority value for dedup: higher = keep.
// Running/paused sessions are preferred over terminal ones.
func (r *Registry) sessionPriority(sess *session.Session) int {
	if sess.IsAlive && !sess.IsTerminal() {
		return 2 // active session — highest priority
	}
	if sess.IsAlive {
		return 1
	}
	return 0 // dead or terminal
}

// removeSessionFiles removes a session's JSON and log files.
func (r *Registry) removeSessionFiles(sess *session.Session) {
	if sess.LogFile != "" {
		os.Remove(sess.LogFile)
	}
	if sess.FilePath != "" {
		os.Remove(sess.FilePath)
	}
}

// CleanupExpired removes all session files older than SessionTTL.
// Called at startup of ralph-admin.
func (r *Registry) CleanupExpired() (int, error) {
	entries, err := os.ReadDir(r.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("reading registry dir: %w", err)
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(r.Dir, entry.Name())
		sess, err := r.readSession(filePath)
		if err != nil {
			os.Remove(filePath)
			removed++
			continue
		}

		if sess.IsExpired() {
			os.Remove(filePath)
			removed++
		}
	}

	return removed, nil
}

// GetSession reads a single session by its ID.
func (r *Registry) GetSession(sessionID string) (*session.Session, error) {
	filePath := filepath.Join(r.Dir, sessionID+".json")
	sess, err := r.readSession(filePath)
	if err != nil {
		return nil, err
	}
	sess.SessionID = sessionID
	sess.FilePath = filePath
	sess.IsAlive = isProcessAlive(sess.PID)
	return sess, nil
}

// KillSession sends SIGTERM to the ralph process.
// Ralph's trap handler will update the session status to "interrupted".
func (r *Registry) KillSession(sess *session.Session) error {
	if sess.IsTerminal() {
		return fmt.Errorf("session is already in terminal state: %s", sess.Status)
	}
	if !sess.IsAlive {
		// Process is dead but session wasn't marked as terminal — mark it
		r.updateStatus(sess, session.StatusDead)
		return nil
	}

	// Send SIGTERM — ralph's trap handler will set status to "interrupted"
	err := syscall.Kill(sess.PID, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("sending SIGTERM to PID %d: %w", sess.PID, err)
	}
	return nil
}

// PauseSession sends SIGSTOP to freeze the ralph process and its children.
func (r *Registry) PauseSession(sess *session.Session) error {
	if !sess.IsAlive {
		return fmt.Errorf("session PID %d is not alive", sess.PID)
	}

	// Send SIGSTOP to the process group (negative PID)
	// This stops the ralph script and its child AI tool process
	err := syscall.Kill(-sess.PID, syscall.SIGSTOP)
	if err != nil {
		// Fallback: try stopping just the process
		err = syscall.Kill(sess.PID, syscall.SIGSTOP)
		if err != nil {
			return fmt.Errorf("sending SIGSTOP to PID %d: %w", sess.PID, err)
		}
	}

	// Update status in session file
	r.updateStatus(sess, session.StatusPaused)
	return nil
}

// ResumeSession sends SIGCONT to unfreeze the ralph process.
func (r *Registry) ResumeSession(sess *session.Session) error {
	if !sess.IsAlive {
		return fmt.Errorf("session PID %d is not alive", sess.PID)
	}

	// Send SIGCONT to the process group
	err := syscall.Kill(-sess.PID, syscall.SIGCONT)
	if err != nil {
		// Fallback: try resuming just the process
		err = syscall.Kill(sess.PID, syscall.SIGCONT)
		if err != nil {
			return fmt.Errorf("sending SIGCONT to PID %d: %w", sess.PID, err)
		}
	}

	// Update status in session file
	r.updateStatus(sess, session.StatusRunning)
	return nil
}

// readSession reads and parses a session JSON file.
func (r *Registry) readSession(filePath string) (*session.Session, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading session file %s: %w", filePath, err)
	}

	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("parsing session file %s: %w", filePath, err)
	}

	return &sess, nil
}

// updateStatus updates the status field in a session's JSON file.
func (r *Registry) updateStatus(sess *session.Session, status string) {
	data, err := os.ReadFile(sess.FilePath)
	if err != nil {
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return
	}

	raw["status"] = status
	updated, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(sess.FilePath, updated, 0644)
	sess.Status = status
}

// isProcessAlive checks if a process with the given PID exists.
// Uses kill(pid, 0) which checks existence without sending a signal.
func isProcessAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

// resolveRealPath resolves symlinks to get the real path.
// Falls back to the original path if resolution fails.
func resolveRealPath(path string) string {
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return real
}

// ReadProgressFile reads the last N lines of the progress.txt file
// from a session's work directory.
func ReadProgressFile(workDir string, maxLines int) (string, error) {
	progressPath := filepath.Join(workDir, ".ralph", "progress.txt")
	data, err := os.ReadFile(progressPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "(no progress file)", nil
		}
		return "", fmt.Errorf("reading progress file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n"), nil
}

// ReadLogTail reads the last maxBytes bytes from the session log file
// and returns the last maxLines lines from that chunk.
// This is efficient for large log files — we only read the tail.
func ReadLogTail(logFile string, maxBytes int64, maxLines int) (string, error) {
	if logFile == "" {
		return "(no log file configured)", nil
	}

	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "(log file not found)", nil
		}
		return "", fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat log file: %w", err)
	}

	size := stat.Size()
	if size == 0 {
		return "(empty log)", nil
	}

	// Read only the tail
	readSize := maxBytes
	if size < readSize {
		readSize = size
	}

	offset := size - readSize
	buf := make([]byte, readSize)
	_, err = f.ReadAt(buf, offset)
	if err != nil {
		return "", fmt.Errorf("reading log tail: %w", err)
	}

	// Split into lines and take last N
	content := string(buf)
	lines := strings.Split(content, "\n")

	// If we started mid-line (offset > 0), drop the first partial line
	if offset > 0 && len(lines) > 0 {
		lines = lines[1:]
	}

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n"), nil
}
