package adapters

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DbConnection struct {
	Name     string
	Host     string
	Port     string
	Username string
	Password string
	Driver   string
	Command  string
	Url      string
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

	return fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s",
		c.Username, c.Password, c.Host, c.Port, database)
}

func (c *DbConnection) InitConnection() (Database, error) {
	var db *sql.DB
	var err error

	db, err = sql.Open(c.Driver, c.String("postgres"))
	if err != nil {
		return nil, err
	}

	var greeting string
	err = db.QueryRow("select 'Hello, world!'").Scan(&greeting)
	if err != nil {
		return nil, err
	}
	if c.Driver == "pgx" {
		return InitPostgres(c), nil
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
