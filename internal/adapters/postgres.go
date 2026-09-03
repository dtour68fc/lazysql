package adapters

import (
	"database/sql"
	"fmt"

	"app.lazygit/internal/session_manager"
)

type Postgres struct {
	dbConnection    *DbConnection
	db              *sql.DB
	currentDatabase string
	sessionManager  *session_manager.SessionManager
}

func InitPostgres(dbConnection *DbConnection) *Postgres {
	// If the connection specifies a target Database, start there instead
	// of the default admin "postgres" db - otherwise RunQuery() (which
	// always targets whatever currentDatabase currently is) would keep
	// hitting "postgres" until something else happened to redirect it,
	// silently running every query against the wrong database.
	currentDatabase := "postgres"
	if dbConnection.Database != "" {
		currentDatabase = dbConnection.Database
	}
	return &Postgres{
		dbConnection:    dbConnection,
		db:              nil,
		currentDatabase: currentDatabase,
		sessionManager:  session_manager.InitSessionManager(),
	}
}

func (p *Postgres) execute(database string, query string, params ...any) (*sql.Rows, error) {
	var err error

	if p.db != nil && p.currentDatabase != database {
		p.db.Close()
		p.db = nil
	}

	if p.db == nil {
		p.db, err = sql.Open(p.dbConnection.Driver, p.dbConnection.String(database))

		if err != nil {
			return nil, err
		}
		p.currentDatabase = database
	}

	result, queryErr := p.db.Query(query, params...)

	if queryErr != nil {
		p.sessionManager.CurrentSession().CreateLog(fmt.Sprintf("Executing query on database '%s': %s, params: %v, error: %v", database, query, params, queryErr.Error()))
	} else {
		p.sessionManager.CurrentSession().CreateLog(fmt.Sprintf("Executing query on database '%s': %s, params: %v", database, query, params))
	}

	if queryErr != nil {
		return nil, queryErr
	}

	return result, queryErr
}

func (p *Postgres) GetDatabases() ([]string, error) {
	// pg_database is a global catalog visible from any database you can
	// connect to - hardcoding "postgres" here meant a user scoped to just
	// their own database (no grant to even connect to "postgres" at all)
	// got denied trying to list databases, even though listing them
	// never actually required connecting to "postgres" specifically.
	rows, err := p.execute(p.currentDatabase, "SELECT datname FROM pg_database WHERE NOT datistemplate;")
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

func (p *Postgres) GetTables(database string) ([]string, error) {
	rows, err := p.execute(database, "SELECT table_name FROM information_schema.tables WHERE table_catalog = $1 AND table_schema NOT IN ('pg_catalog', 'information_schema');", database)
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

func (p *Postgres) GetTableItem(database string, table string, item string) ([][]string, error) {
	var rows *sql.Rows
	var err error
	switch item {
	case "data":
		rows, err = p.execute(database, fmt.Sprintf("SELECT * FROM %s LIMIT 100;", table))
	case "schema":
		rows, err = p.execute(database, "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1", table)
	case "indexes":
		rows, err = p.execute(database, "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = $1", table)
	default:
		return nil, fmt.Errorf("unknown item: %s", item)
	}

	if err != nil {
		return nil, err
	}
	return p.InspectRows(rows)
}

func (p *Postgres) RunQuery(query string) ([][]string, error) {
	rows, err := p.execute(p.currentDatabase, query)
	if err != nil {
		return nil, err
	}
	return p.InspectRows(rows)
}

// CurrentDatabaseForTest exposes currentDatabase for tests confirming
// InitPostgres seeds it correctly from DbConnection.Database.
func (p *Postgres) CurrentDatabaseForTest() string {
	return p.currentDatabase
}

func (p *Postgres) InspectRows(rows *sql.Rows) ([][]string, error) {
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
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
