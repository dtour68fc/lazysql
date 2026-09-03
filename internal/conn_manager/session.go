package conn_manager

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SessionState is the last-connected project/database/table, saved every
// time you connect to a project, land on a specific database, or open a
// table, and restored automatically on the next launch - so quitting and
// reopening drops you right back where you left off instead of starting
// from the Projects tab every time.
type SessionState struct {
	ProjectName  string
	DatabaseName string
	TableName    string
}

func getSessionFilePath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lazysql", "session.json"), nil
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(userConfigDir, "lazysql", "session.json"), nil
}

func saveSession(s SessionState) error {
	sessionPath, err := getSessionFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(sessionPath), os.ModePerm); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath, data, 0644)
}

// loadSession returns the zero SessionState (not an error) if there's no
// session file yet - a brand new install shouldn't fail to start just
// because it's never saved a session before.
func loadSession() (SessionState, error) {
	sessionPath, err := getSessionFilePath()
	if err != nil {
		return SessionState{}, err
	}
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionState{}, nil
		}
		return SessionState{}, err
	}
	var s SessionState
	if err := json.Unmarshal(data, &s); err != nil {
		return SessionState{}, err
	}
	return s, nil
}
