package adapters

import (
	"fmt"
	"os"
	"strings"
)

func RawQueryLoggingEnabled() bool {
	value := strings.ToLower(os.Getenv("LAZYSQL_LOG_QUERIES"))
	return value == "1" || value == "true" || value == "yes"
}

func formatQueryLog(database string, query string, params []any, queryErr error) string {
	if RawQueryLoggingEnabled() {
		if queryErr != nil {
			return fmt.Sprintf("Executing query on database '%s': %s, params: %v, error: %v", database, query, params, queryErr.Error())
		}
		return fmt.Sprintf("Executing query on database '%s': %s, params: %v", database, query, params)
	}

	if queryErr != nil {
		return fmt.Sprintf("Executing query on database '%s': <redacted>, params_count: %d, error: %v", database, len(params), queryErr.Error())
	}
	return fmt.Sprintf("Executing query on database '%s': <redacted>, params_count: %d", database, len(params))
}
