package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDbConnectionString(t *testing.T) {
	t.Run("basic connection string", func(t *testing.T) {
		c := DbConnection{
			Username: "postgres",
			Password: "secret",
			Host:     "localhost",
			Port:     "5432",
		}
		got := c.String("testdb")
		want := "user=postgres password=secret host=localhost port=5432 database=testdb"
		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("connection string from URL", func(t *testing.T) {
		c := DbConnection{
			Url: "postgres://user:pass@remotehost:5433/ignored",
		}
		got := c.String("overridedb")
		want := "user=user password=pass host=remotehost port=5433 database=overridedb"
		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})

	t.Run("connection string from command", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test-script.sh")
		err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf 'cmdhost\tcmduser\tcmdpass\t9999'"), 0755)
		if err != nil {
			t.Fatalf("failed to write temp script: %v", err)
		}

		c := DbConnection{
			Command: scriptPath,
		}
		got := c.String("cmddb")
		want := "user=cmduser password=cmdpass host=cmdhost port=9999 database=cmddb"
		if got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
}

func TestCollectCredentialsFromCommand(t *testing.T) {
	t.Run("valid command output", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test-script.sh")
		err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf 'h\tu\tp\t1234'"), 0755)
		if err != nil {
			t.Fatalf("failed to write temp script: %v", err)
		}

		c := &DbConnection{
			Command: scriptPath,
		}
		err = c.collectCredentialsFromCommand()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Host != "h" || c.Username != "u" || c.Password != "p" || c.Port != "1234" {
			t.Errorf("unexpected credentials: %+v", c)
		}
	})

	t.Run("invalid command output", func(t *testing.T) {
		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test-script.sh")
		err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nprintf 'only\tthree\tparts'"), 0755)
		if err != nil {
			t.Fatalf("failed to write temp script: %v", err)
		}

		c := &DbConnection{
			Command: scriptPath,
		}
		err = c.collectCredentialsFromCommand()
		if err == nil {
			t.Error("expected error for invalid output, got nil")
		}
	})

	t.Run("command execution failure", func(t *testing.T) {
		c := &DbConnection{
			Command: "nonexistent-command-12345",
		}
		err := c.collectCredentialsFromCommand()
		if err == nil {
			t.Error("expected error for nonexistent command, got nil")
		}
	})
}
