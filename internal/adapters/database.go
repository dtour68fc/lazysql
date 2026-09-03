package adapters

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/go-sql-driver/mysql"
)

// connectTimeout bounds how long InitConnection's health-check query is
// allowed to hang waiting on a slow/unreachable host. Without this, a
// dead/unreachable connection just hangs the "Connecting..." UI forever
// with no error at all - the underlying database/sql query had no
// context/deadline on it whatsoever.
const connectTimeout = 8 * time.Second

type DbConnection struct {
	Name     string
	Host     string
	Port     string
	Username string
	Password string
	Driver   string
	Command  string
	Url      string
	// Database is the specific database name this connection's tables
	// should be listed from (e.g. "pmo_db"). Optional - if empty, the
	// Tables tab falls back to whichever database GetDatabases() happens
	// to return first, which is often just the empty admin "postgres"/
	// "mysql" database, not the one the user actually cares about.
	Database string
	// No separate Project field - a connection's Name IS its project alias
	// (e.g. "PMO" -> localhost:5432), so a second optional grouping tag was
	// redundant once the Connection Manager stopped supporting multiple
	// connections per project.
}

type Database interface {
	GetDatabases() ([]string, error)
	GetTables(string) ([]string, error)
	GetTableItem(string, string, string) ([][]string, error)
	RunQuery(string) ([][]string, error)
}

func (c *DbConnection) String(database string) string {
	if c.Url != "" {
		u, err := url.Parse(c.Url)
		if err == nil {
			c.Username = u.User.Username()
			c.Password, _ = u.User.Password()
			c.Host = u.Hostname()
			c.Port = u.Port()
		}
	} else if c.Command != "" {
		c.collectCredentialsFromCommand()
	}

	if c.Driver == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", c.Username, c.Password, c.Host, c.Port, database)
	}

	return fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s",
		c.Username, c.Password, c.Host, c.Port, database)
}

func (c *DbConnection) InitConnection() (Database, error) {
	var db *sql.DB
	var err error
	var database string

	if c.Driver == "pgx" {
		database = "postgres"
	} else if c.Driver == "mysql" {
		database = "mysql"
	} else {
		return nil, fmt.Errorf("unsupported driver: %s", c.Driver)
	}

	db, err = sql.Open(c.Driver, c.String(database))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	var greeting string
	err = db.QueryRowContext(ctx, "select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		db.Close()
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("timed out connecting to %s:%s after %s (host unreachable or too slow to respond)", c.Host, c.Port, connectTimeout)
		}
		return nil, err
	}
	if c.Driver == "pgx" {
		return InitPostgres(c), nil
	} else if c.Driver == "mysql" {
		return InitMySQL(c), nil
	}
	defer db.Close()
	return nil, fmt.Errorf("unsupported driver: %s", c.Driver)
}

func (c *DbConnection) collectCredentialsFromCommand() error {
	out, err := exec.Command(c.Command).Output()
	if err != nil {
		return err
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(parts) != 4 {
		return fmt.Errorf("invalid command output: expected 4 tab-separated values, got %d", len(parts))
	}
	c.Host = parts[0]
	c.Username = parts[1]
	c.Password = parts[2]
	c.Port = parts[3]
	return nil
}
