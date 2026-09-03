package conn_manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	adapters "app.lazygit/internal/adapters"
)

// dumpQuote escapes single quotes for a naive inline SQL literal - matches
// the same simple approach the viewer's cell-edit UPDATE builder uses (see
// internal/viewer/viewer.go's sqlQuote) - good enough for generating a
// dump file, not a substitute for real parameterized queries.
func dumpQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// tableInsertStatements runs "SELECT * FROM table" and turns every row
// into an INSERT INTO statement - a data-only dump. There's no DDL/schema
// introspection here (no CREATE TABLE), just re-insertable data - if you
// need the schema too, this is only half of a real pg_dump/mysqldump.
func tableInsertStatements(database adapters.Database, table string) ([]string, error) {
	rows, err := database.RunQuery(fmt.Sprintf("SELECT * FROM %s;", table))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	columns := rows[0]
	var statements []string
	for _, row := range rows[1:] {
		values := make([]string, len(row))
		for i, v := range row {
			values[i] = dumpQuote(v)
		}
		statements = append(statements, fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s);",
			table, strings.Join(columns, ", "), strings.Join(values, ", "),
		))
	}
	return statements, nil
}

// dumpTable writes a single table's data (data-only, see
// tableInsertStatements) to <folder>/<table>.sql.
func dumpTable(database adapters.Database, table string, folder string) (string, error) {
	statements, err := tableInsertStatements(database, table)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	body.WriteString(fmt.Sprintf("-- Dump of table %s\n\n", table))
	for _, stmt := range statements {
		body.WriteString(stmt)
		body.WriteString("\n")
	}
	path := filepath.Join(folder, table+".sql")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(body.String()), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// dumpDatabase writes every table's data (data-only, same caveat as
// dumpTable) in a database to a single <folder>/<database>.sql file, one
// section per table.
func dumpDatabase(database adapters.Database, databaseName string, folder string) (string, error) {
	tables, err := database.GetTables(databaseName)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	body.WriteString(fmt.Sprintf("-- Dump of database %s\n", databaseName))
	for _, table := range tables {
		statements, err := tableInsertStatements(database, table)
		if err != nil {
			return "", fmt.Errorf("table %s: %w", table, err)
		}
		body.WriteString(fmt.Sprintf("\n-- Table: %s\n\n", table))
		for _, stmt := range statements {
			body.WriteString(stmt)
			body.WriteString("\n")
		}
	}
	path := filepath.Join(folder, databaseName+".sql")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(body.String()), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// importSQLFile re-runs a previously dumped .sql file (table or database,
// same round-trippable data-only format written by dumpTable/dumpDatabase
// above) against the given database - one statement per non-comment,
// non-blank line, matching exactly how those write one complete
// "INSERT INTO ...;" per line. Stops on the first failing statement rather
// than silently skipping it, reporting how many ran successfully before
// that happened.
func importSQLFile(database adapters.Database, path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	ran := 0
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		if _, err := database.RunQuery(line); err != nil {
			return ran, fmt.Errorf("statement %d: %w", ran+1, err)
		}
		ran++
	}
	return ran, nil
}
