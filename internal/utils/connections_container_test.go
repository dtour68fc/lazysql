package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConnections(t *testing.T) {
	// Setup temporary config directory
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Test Case 1: File doesn't exist
	connections, err := GetConnections()
	if err != nil {
		t.Fatalf("Expected no error when file doesn't exist, got %v", err)
	}
	if len(connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(connections))
	}

	// Verify file was created
	configPath := filepath.Join(tempDir, "lazysql", "connections.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Expected connections.json to be created")
	}

	// Test Case 2: File exists with connections
	// We need to write to the file manually to test reading it back
	// but wait, we don't have a SaveConnections function in utils yet.
	// So we'll just write it directly.
	content := `{"test": {"Name": "test", "Host": "localhost"}}`
	err = os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	connections, err = GetConnections()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connections))
	}
	if connections["test"].Name != "test" {
		t.Errorf("Expected connection name 'test', got '%s'", connections["test"].Name)
	}

	// Test Case 3: Invalid JSON
	err = os.WriteFile(configPath, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	_, err = GetConnections()
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestGetConnections_CreateDirFail(t *testing.T) {
	// Setup a path that should fail to be created
	// We can't easily mock os.MkdirAll, but we can point it to a read-only directory
	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0555) // Read and execute only
	if err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	// We want GetConnections to try and create a subdir in readOnlyDir
	// getConnectionsFilePath returns userConfigDir + "/lazysql/connections.json"
	// if we set XDG_CONFIG_HOME to readOnlyDir, it will try to create readOnlyDir/lazysql
	t.Setenv("XDG_CONFIG_HOME", readOnlyDir)

	_, err = GetConnections()
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
