package utils

import (
	"encoding/json"
	"os"
	"strings"

	"app.lazygit/internal/adapters"
)

func GetConnections() (map[string]adapters.DbConnection, error) {
	fileContent, err := readConnectionsFile()
	if err != nil {
		return nil, err
	}

	var connections map[string]adapters.DbConnection
	err = json.Unmarshal(fileContent, &connections)
	if err != nil {
		return nil, err
	}

	return connections, nil
}

func readConnectionsFile() ([]byte, error) {
	var fileContent []byte
	var fileErr error

	connectionsPath, err := getConnectionsFilePath()
	lazysqlConfigDir := strings.ReplaceAll(connectionsPath, "/connections.json", "")
	if err != nil {
		return fileContent, err
	}

	fileContent, fileErr = os.ReadFile(connectionsPath)
	if fileErr != nil {
		if os.IsNotExist(fileErr) {
			fileErr = nil
			os.MkdirAll(lazysqlConfigDir+"/lazysql", os.ModePerm)
			os.WriteFile(connectionsPath, []byte("{}"), 0644)
			fileContent, fileErr = os.ReadFile(connectionsPath)
		} else {
			return fileContent, fileErr
		}
	}
	return fileContent, nil
}

func getConnectionsFilePath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return userConfigDir + "/lazysql/connections.json", nil
}
