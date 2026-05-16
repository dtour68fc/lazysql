package session_manager

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionManager_PresentLogic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazysql-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := &SessionManager{
		configDir: tempDir,
	}

	// 1. Test getSessionsDir logic
	sessionsDir := s.getSessionsDir()
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		t.Error("Expected getSessionsDir to create the directory")
	}

	// 2. Test getCreatedSessions logic
	// Create a mock session log file
	pid := 12345
	pidFile := filepath.Join(sessionsDir, fmt.Sprintf("session-%d.log", pid))
	os.WriteFile(pidFile, []byte{}, 0644)

	pids := s.getCreatedSessions()
	if len(pids) != 1 || pids[0] != pid {
		t.Errorf("Expected pids [12345], got %v", pids)
	}

	// 3. Test activeSessions logic
	// Our mock PID 12345 is likely not running
	active := s.activeSessions()
	for _, a := range active {
		if a == pid {
			t.Errorf("PID %d should not be active", pid)
		}
	}

	// 4. Test removeTerminatedSessions logic (as implemented)
	s.removeTerminatedSessions()

	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Errorf("Expected %s to be removed by removeTerminatedSessions", pidFile)
	}
}
