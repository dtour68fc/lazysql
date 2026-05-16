package conn_manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app.lazygit/internal/adapters"
	"github.com/zalando/go-keyring"
)

func TestGetConnections(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	connections, err := getConnections()
	if err != nil {
		t.Fatalf("Expected no error when file doesn't exist, got %v", err)
	}
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}

	configPath := filepath.Join(tempDir, "lazysql", "connections.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected connections.json to be created")
	}

	content := `{"test": {"Name": "test", "Host": "localhost"}}`
	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	connections, err = getConnections()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connections))
	}
	if connections["test"].Name != "test" {
		t.Errorf("Expected connection name 'test', got '%s'", connections["test"].Name)
	}

	err = os.WriteFile(configPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	_, err = getConnections()
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSaveConnections(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_, _ = getConnections()

	connections := map[string]adapters.DbConnection{
		"test-save": {
			Name: "test-save",
			Host: "localhost",
		},
	}

	err := saveConnections(connections)
	if err != nil {
		t.Fatalf("saveConnections failed: %v", err)
	}

	saved, err := getConnections()
	if err != nil {
		t.Fatalf("getConnections failed: %v", err)
	}

	if len(saved) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(saved))
	}

	if saved["test-save"].Name != "test-save" {
		t.Errorf("Expected connection name 'test-save', got '%s'", saved["test-save"].Name)
	}
}

func TestKeyringIntegration(t *testing.T) {
	keyring.MockInit()

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_, _ = getConnections()

	connections := map[string]adapters.DbConnection{
		"secure-conn": {
			Name:     "secure-conn",
			Password: "secret-password",
			Url:      "postgres://user:url-pass@localhost:5432/db",
		},
	}

	err := saveConnections(connections)
	if err != nil {
		t.Fatalf("saveConnections failed: %v", err)
	}

	configPath := filepath.Join(tempDir, "lazysql", "connections.json")
	content, _ := os.ReadFile(configPath)

	if strings.Contains(string(content), "secret-password") {
		t.Error("Plain text password found in connections.json")
	}

	if strings.Contains(string(content), "url-pass") {
		t.Error("Plain text URL password found in connections.json")
	}

	restored, err := getConnections()
	if err != nil {
		t.Fatalf("getConnections failed: %v", err)
	}

	if restored["secure-conn"].Password != "secret-password" {
		t.Errorf("Expected password 'secret-password', got '%s'", restored["secure-conn"].Password)
	}
	if restored["secure-conn"].Url != "postgres://user:url-pass@localhost:5432/db" {
		t.Errorf("Expected URL with password, got '%s'", restored["secure-conn"].Url)
	}

	deleteFromKeyring("secure-conn")
}

func TestReadConnectionsFile_OtherError(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	configPath := filepath.Join(tempDir, "lazysql", "connections.json")
	err := os.MkdirAll(configPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err = getConnections()
	if err == nil {
		t.Error("Expected error when connections.json is a directory, got nil")
	}
}

func TestGetConnections_CreateDirFail(t *testing.T) {
	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0555)
	if err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", readOnlyDir)

	_, err = getConnections()
	if err == nil {
		t.Error("Expected error when directory creation fails, got nil")
	}
}

func TestGetConnectionsFilePath(t *testing.T) {
	tempDir := "/tmp/test-config"
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	path, err := getConnectionsFilePath()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := filepath.Join(tempDir, "lazysql", "connections.json")
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

func TestGetConnectionsFilePath_Error(t *testing.T) {
	home := os.Getenv("HOME")
	os.Unsetenv("HOME")
	defer os.Setenv("HOME", home)

	_, err := getConnectionsFilePath()
	if err == nil {
		t.Error("Expected error when HOME is unset, got nil")
	}
}
