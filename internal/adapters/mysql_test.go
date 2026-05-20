package adapters

import (
	"database/sql"
	"fmt"
	"testing"
)

func TestMySQLGetDatabases(t *testing.T) {
	dbConnection := DbConnection{
		Name:     "testmysql",
		Host:     "localhost",
		Port:     "3306",
		Username: "root",
		Password: "mysql",
		Driver:   "mysql",
	}

	database, err := dbConnection.InitConnection()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	mysql := database.(*Mysql)
	result, err := mysql.GetDatabases()
	if err != nil {
		t.Fatalf("Failed to get databases: %v", err)
	}
	if len(result) == 0 {
		t.Fatalf("Expected at least one database, got 0")
	}
}

func TestMySQLGetTables(t *testing.T) {
	dbConnection := DbConnection{
		Name:     "testmysql",
		Host:     "localhost",
		Port:     "3306",
		Username: "root",
		Password: "mysql",
		Driver:   "mysql",
	}

	database, err := dbConnection.InitConnection()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	mysql := database.(*Mysql)
	result, err := mysql.GetTables("mysql")
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}
	if len(result) == 0 {
		t.Fatalf("Expected at least one table, got 0")
	}
}

func TestMySQLGetTableItem(t *testing.T) {
	const testDBName = "lazysql_test_mysql"
	const testTable = "users"

	dbConnection := DbConnection{
		Name:     "testmysql",
		Host:     "localhost",
		Port:     "3306",
		Username: "root",
		Password: "mysql",
		Driver:   "mysql",
	}

	database, err := dbConnection.InitConnection()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	mysql := database.(*Mysql)

	adminDB, err := sql.Open(dbConnection.Driver, dbConnection.String("mysql"))
	if err != nil {
		t.Fatalf("Failed to open admin connection: %v", err)
	}
	defer adminDB.Close()

	adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName)); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
	})

	testDB, err := sql.Open(dbConnection.Driver, dbConnection.String(testDBName))
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	if _, err := testDB.Exec(fmt.Sprintf(`
		CREATE TABLE %s (
			id    INT AUTO_INCREMENT PRIMARY KEY,
			name  VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		)`, testTable)); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	if _, err := testDB.Exec(fmt.Sprintf(
		"INSERT INTO %s (name, email) VALUES ('Alice', 'alice@example.com'), ('Bob', NULL)",
		testTable,
	)); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Run("data", func(t *testing.T) {
		result, err := mysql.GetTableItem(testDBName, testTable, "data")
		if err != nil {
			t.Fatalf("GetTableItem(data) failed: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("Expected 3 rows (1 header + 2 data), got %d", len(result))
		}
		if len(result[0]) != 3 {
			t.Errorf("Expected 3 columns (id, name, email), got %d", len(result[0]))
		}
	})

	t.Run("schema", func(t *testing.T) {
		result, err := mysql.GetTableItem(testDBName, testTable, "schema")
		if err != nil {
			t.Fatalf("GetTableItem(schema) failed: %v", err)
		}
		if len(result) != 4 {
			t.Fatalf("Expected 4 rows (1 header + 3 columns), got %d", len(result))
		}
		headers := result[0]
		if (headers[0] != "column_name" && headers[0] != "COLUMN_NAME") || (headers[1] != "data_type" && headers[1] != "DATA_TYPE") {
			t.Errorf("Unexpected schema headers: %v", headers)
		}
	})

	t.Run("indexes", func(t *testing.T) {
		result, err := mysql.GetTableItem(testDBName, testTable, "indexes")
		if err != nil {
			t.Fatalf("GetTableItem(indexes) failed: %v", err)
		}
		if len(result) < 2 {
			t.Fatalf("Expected at least 2 rows (1 header + 1 index), got %d", len(result))
		}
		headers := result[0]
		foundTable := false
		for i, h := range headers {
			if h == "Table" {
				foundTable = true
				if result[1][i] != testTable {
					t.Errorf("Expected table name %s, got %s", testTable, result[1][i])
				}
				break
			}
		}
		if !foundTable {
			t.Errorf("Unexpected indexes headers (missing 'Table'): %v", headers)
		}
	})

	t.Run("unknown item returns error", func(t *testing.T) {
		_, err := mysql.GetTableItem(testDBName, testTable, "unknown")
		if err == nil {
			t.Error("Expected an error for unknown item type, got nil")
		}
	})
}
