package conn_manager

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"

	"app.lazygit/internal/adapters"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "lazysql"
)

func saveConnections(connections map[string]adapters.DbConnection) error {
	connectionsPath, err := getConnectionsFilePath()
	if err != nil {
		return err
	}

	saveMap := make(map[string]adapters.DbConnection)
	for name, conn := range connections {
		connCopy := conn

		if connCopy.Password != "" {
			err := keyring.Set(keyringService, name+"-password", connCopy.Password)
			if err == nil {
				connCopy.Password = ""
			}
		}

		if connCopy.Url != "" {
			u, err := url.Parse(connCopy.Url)
			if err == nil && u.User != nil {
				_, hasPass := u.User.Password()
				if hasPass {
					err := keyring.Set(keyringService, name+"-url", connCopy.Url)
					if err == nil {
						connCopy.Url = ""
					}
				}
			}
		}
		saveMap[name] = connCopy
	}

	connectionsJson, err := json.MarshalIndent(saveMap, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(connectionsPath, connectionsJson, 0644)
	if err != nil {
		return err
	}

	return nil
}

func initializeNewConnection() adapters.DbConnection {
	return adapters.DbConnection{
		Name:     "New Connection",
		Host:     "",
		Port:     "",
		Username: "",
		Password: "",
		Driver:   "",
		Command:  "",
		Url:      "",
	}
}

func getConnections() (map[string]adapters.DbConnection, error) {
	fileContent, err := readConnectionsFile()
	if err != nil {
		return nil, err
	}

	var connections map[string]adapters.DbConnection
	err = json.Unmarshal(fileContent, &connections)
	if err != nil {
		return nil, err
	}

	for name, conn := range connections {
		if pw, err := keyring.Get(keyringService, name+"-password"); err == nil {
			conn.Password = pw
		}

		if u, err := keyring.Get(keyringService, name+"-url"); err == nil {
			conn.Url = u
		}
		connections[name] = conn
	}

	return connections, nil
}

func deleteFromKeyring(name string) error {
	keyring.Delete(keyringService, name+"-password")
	keyring.Delete(keyringService, name+"-url")
	return nil
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
