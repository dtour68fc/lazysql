package adapters

import (
	"database/sql"
	"fmt"

	"app.lazygit/internal/session_manager"
)

type Mysql struct {
	dbConnection    *DbConnection
	db              *sql.DB
	currentDatabase string
	sessionManager  *session_manager.SessionManager
}

func InitMySQL(dbConnection *DbConnection) *Mysql {
	// See InitPostgres for why this defaults to dbConnection.Database when
	// set, instead of always starting at the admin "mysql" db.
	currentDatabase := "mysql"
	if dbConnection.Database != "" {
		currentDatabase = dbConnection.Database
	}
	return &Mysql{
		dbConnection:    dbConnection,
		db:              nil,
		currentDatabase: currentDatabase,
		sessionManager:  session_manager.InitSessionManager(),
	}
}

func (m *Mysql) execute(database string, query string, params ...any) (*sql.Rows, error) {
	var err error

	if m.db != nil && m.currentDatabase != database {
		m.db.Close()
		m.db = nil
	}

	if m.db == nil {
		m.db, err = sql.Open(m.dbConnection.Driver, m.dbConnection.String(database))

		if err != nil {
			return nil, err
		}
		m.currentDatabase = database
	}

	result, queryErr := m.db.Query(query, params...)

	if queryErr != nil {
		 m.sessionManager.CurrentSession().CreateLog(fmt.Sprintf("Executing query on database '%s': %s, params: %v, error: %v", database, query, params, queryErr.Error()))
	} else {
		 m.sessionManager.CurrentSession().CreateLog(fmt.Sprintf("Executing query on database '%s': %s, params: %v", database, query, params))
	}


	if queryErr != nil {
		return nil, queryErr
	}

	return result, queryErr
}

func (m *Mysql) GetDatabases() ([]string, error) {
	rows, err := m.execute("mysql", "SHOW DATABASES;")
	if err != nil {
		return nil, err
	}
	var databases []string
	for rows.Next() {
		var datname string
		if err := rows.Scan(&datname); err != nil {
			return nil, err
		}
		databases = append(databases, datname)
	}

	return databases, rows.Err()
}

func (m *Mysql) GetTables(database string) ([]string, error) {
	rows, err := m.execute(database, "SELECT table_name FROM information_schema.tables WHERE table_schema = ?;", database)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var tablename string
		if err := rows.Scan(&tablename); err != nil {
			return nil, err
		}
		tables = append(tables, tablename)
	}

	return tables, rows.Err()
}

func (m *Mysql) GetTableItem(database string, table string, item string) ([][]string, error) {
	var rows *sql.Rows
	var err error
	switch item {
	case "data":
		rows, err = m.execute(database, fmt.Sprintf("SELECT * FROM `%s` LIMIT 100;", table))
	case "schema":
		rows, err = m.execute(database, "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = ? AND table_schema = ?", table, database)
	case "indexes":
		rows, err = m.execute(database, fmt.Sprintf("SHOW INDEX FROM `%s`", table))
	default:
		return nil, fmt.Errorf("unknown item: %s", item)
	}

	if err != nil {
		return nil, err
	}
	return m.InspectRows(rows)
}

func (m *Mysql) RunQuery(query string) ([][]string, error) {
	rows, err := m.execute(m.currentDatabase, query)
	if err != nil {
		return nil, err
	}
	return m.InspectRows(rows)
}

func (m *Mysql) InspectRows(rows *sql.Rows) ([][]string, error) {
	if rows == nil {
		return nil, fmt.Errorf("rows is nil")
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := [][]string{columns}

	vals := make([]any, len(columns))
	valPtrs := make([]any, len(columns))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		row := make([]string, len(columns))
		for i, val := range vals {
			if val == nil {
				row[i] = ""
			} else if b, ok := val.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
