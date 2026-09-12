package adapters

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatQueryLogRedactsByDefault(t *testing.T) {
	got := formatQueryLog("prod", "SELECT * FROM users WHERE email = $1", []any{"person@example.com"}, nil)

	if strings.Contains(got, "SELECT") {
		t.Fatalf("expected query text to be redacted, got %q", got)
	}
	if strings.Contains(got, "person@example.com") {
		t.Fatalf("expected query params to be redacted, got %q", got)
	}
	if !strings.Contains(got, "<redacted>") || !strings.Contains(got, "params_count: 1") {
		t.Fatalf("expected redacted log with params count, got %q", got)
	}
}

func TestFormatQueryLogRawOptIn(t *testing.T) {
	t.Setenv("LAZYSQL_LOG_QUERIES", "1")

	got := formatQueryLog("prod", "SELECT * FROM users WHERE email = $1", []any{"person@example.com"}, nil)

	if !strings.Contains(got, "SELECT * FROM users") {
		t.Fatalf("expected query text when raw logging is enabled, got %q", got)
	}
	if !strings.Contains(got, "person@example.com") {
		t.Fatalf("expected query params when raw logging is enabled, got %q", got)
	}
}

func TestFormatQueryLogKeepsErrorWithoutRawQuery(t *testing.T) {
	got := formatQueryLog("prod", "DROP TABLE users", nil, errors.New("permission denied"))

	if strings.Contains(got, "DROP TABLE") {
		t.Fatalf("expected query text to be redacted, got %q", got)
	}
	if !strings.Contains(got, "permission denied") {
		t.Fatalf("expected error to remain useful, got %q", got)
	}
}
