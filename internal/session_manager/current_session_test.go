package session_manager

import (
	"os"
	"strings"
	"testing"
)

func TestCurrentSession(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazysql-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Test Initialization
	cs := InitCurrentSession(tempDir)
	if cs.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), cs.PID)
	}
	if !strings.Contains(cs.sessionFilePath, tempDir) {
		t.Errorf("Expected session file path to be in %s, got %s", tempDir, cs.sessionFilePath)
	}

	// 2. Test CreateLog (initial creation)
	logContent := "First log entry"
	err = cs.CreateLog(logContent)
	if err != nil {
		t.Errorf("CreateLog failed: %v", err)
	}

	data, err := os.ReadFile(cs.sessionFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if strings.TrimSpace(string(data)) != logContent {
		t.Errorf("Expected content %q, got %q", logContent, string(data))
	}

	// 3. Test CreateLog (appending)
	secondEntry := "Second log entry"
	err = cs.CreateLog(secondEntry)
	if err != nil {
		t.Errorf("CreateLog failed on append: %v", err)
	}

	data, err = os.ReadFile(cs.sessionFilePath)
	if !strings.Contains(string(data), logContent) || !strings.Contains(string(data), secondEntry) {
		t.Errorf("Log file missing content after append. Got: %q", string(data))
	}

	// 4. Test Cleanup
	cs.Cleanup()
	if _, err := os.Stat(cs.sessionFilePath); !os.IsNotExist(err) {
		t.Error("Expected log file to be removed after Cleanup()")
	}
}
