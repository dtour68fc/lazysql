package session_manager

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"syscall"
)

type SessionManager struct {
	configDir      string
	sessionsDir    string
	currentSession *CurrentSession
}

var manager *SessionManager
var once sync.Once

func InitSessionManager() *SessionManager {
	once.Do(func() {
		manager = &SessionManager{}
		manager.removeTerminatedSessions()
	})
	return manager
}

func (s *SessionManager) CurrentSession() *CurrentSession {
	if s.currentSession == nil {
		s.currentSession = InitCurrentSession(s.getSessionsDir())
	}
	return s.currentSession
}

func (s *SessionManager) removeTerminatedSessions() {
	createdSessions := s.getCreatedSessions()
	activeSessions := s.activeSessions()
	for _, pid := range createdSessions {
		if !slices.Contains(activeSessions, pid) {

			pidFile := filepath.Join(s.getSessionsDir(), fmt.Sprintf("session-%d.log", pid))
			os.Remove(pidFile)
		}
	}
}

func (s *SessionManager) getConfigDir() string {
	if s.configDir != "" {
		return s.configDir
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		s.configDir = xdg
		return s.configDir
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		panic("Unable to determine user config directory")
	}
	s.configDir = configDir
	return s.configDir
}

func (s *SessionManager) getSessionsDir() string {
	if s.sessionsDir != "" {
		return s.sessionsDir
	}
	sessionsDir := filepath.Join(s.getConfigDir(), "lazysql", "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		os.MkdirAll(sessionsDir, 0755)
	}
	s.sessionsDir = sessionsDir
	return s.sessionsDir
}

func (s *SessionManager) getCreatedSessions() []int {
	sessionsDir := s.getSessionsDir()
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return []int{}
	}
	var pids []int
	for _, entry := range entries {
		var pid int
		_, err := fmt.Sscanf(entry.Name(), "session-%d.log", &pid)
		if err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

func (s *SessionManager) activeSessions() []int {
	pids := s.getCreatedSessions()
	activePids := []int{}
	for _, pid := range pids {
		process, err := os.FindProcess(pid)
		if err != nil {
			continue
		}
		err = process.Signal(syscall.Signal(0))
		if err == nil {
			activePids = append(activePids, pid)
		}
	}
	return activePids
}
