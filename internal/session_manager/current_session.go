package session_manager

import (
	"fmt"
	"os"
	"path/filepath"
)

type CurrentSession struct {
	sessionsDir     string
	sessionFilePath string
	PID             int
}

func InitCurrentSession(sessionsDir string) *CurrentSession {
	pid := os.Getpid()
	sessionFileName := fmt.Sprintf("session-%d.log", pid)
	sessionFilePath := filepath.Join(sessionsDir, sessionFileName)
	return &CurrentSession{
		sessionsDir:     sessionsDir,
		sessionFilePath: sessionFilePath,
		PID:             pid,
	}
}

func (s *CurrentSession) CreateLog(log string) error {
	f, err := os.OpenFile(s.sessionFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to create session log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(log + "\n"); err != nil {
		return fmt.Errorf("unable to write to session log file: %w", err)
	}
	return nil
}

func (s *CurrentSession) Cleanup() {
	os.Remove(s.sessionFilePath)
}
