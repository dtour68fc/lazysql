package editor

import (
	"fmt"
	"regexp"
	"strings"
)

// parseShorthandQuery turns the vim-shorthand DSL typed in command mode
// into real SQL. Grammar (space-separated, one leading ":" already
// stripped by vimtea before we ever see it, clauses after the first are
// separated by " :"):
//
// Select head (either form), with an optional alias:
//
//	sa <table> [as <alias>]                -> SELECT * FROM <table> [AS <alias>]
//	s(col1,col2,...) <table> [as <alias>]  -> SELECT col1, col2 FROM <table> [AS <alias>]
//
// Chained onto a select head, in any order, any number of times:
//
//	:j <table> [as <alias>] [on <a.col>=<b.col>]   -> JOIN ...
//	:lj <table> [as <alias>] [on <a.col>=<b.col>]  -> LEFT JOIN ...
//	:w <condition>                                 -> WHERE <condition>
//	                                                   (multiple :w clauses AND together)
//
// A join with no explicit "on" condition doesn't guess a foreign key
// (guessing wrong silently is worse than not guessing at all) - it seeds a
// TODO placeholder instead and autoRun comes back false, so you get a
// buffer to finish rather than a query fired off with a broken or
// accidental cross-join condition.
//
// Standalone heads (no chaining with the above - each is a complete
// statement on its own):
//
//	d <table>            -> DELETE FROM <table>
//	d <table> :w <cond>  -> DELETE FROM <table> WHERE <cond> (multiple :w AND together)
//	i <table>(col=val,...) -> INSERT INTO <table> (col, ...) VALUES (val, ...)
func parseShorthandQuery(raw string) (query string, autoRun bool, ok bool) {
	clauses := strings.Split(raw, " :")
	if len(clauses) == 0 {
		return "", false, false
	}
	head := strings.TrimSpace(clauses[0])
	rest := clauses[1:]

	if m := deleteRe.FindStringSubmatch(head); m != nil {
		return parseDeleteClauses(m[1], rest)
	}
	if m := insertRe.FindStringSubmatch(head); m != nil {
		if len(rest) > 0 {
			return "", false, false
		}
		return parseInsert(m[1], m[2])
	}
	return parseSelectClauses(head, rest)
}

func parseDeleteClauses(table string, rest []string) (query string, autoRun bool, ok bool) {
	base := fmt.Sprintf("DELETE FROM %s", table)
	var whereConds []string
	for _, clause := range rest {
		clause = strings.TrimSpace(clause)
		m := whereRe.FindStringSubmatch(clause)
		if m == nil {
			return "", false, false
		}
		whereConds = append(whereConds, m[1])
	}
	if len(whereConds) > 0 {
		base += " WHERE " + strings.Join(whereConds, " AND ")
	}
	return base + ";", true, true
}

func parseInsert(table string, assignments string) (query string, autoRun bool, ok bool) {
	pairs := strings.Split(assignments, ",")
	var cols, vals []string
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return "", false, false
		}
		cols = append(cols, strings.TrimSpace(kv[0]))
		vals = append(vals, strings.TrimSpace(kv[1]))
	}
	if len(cols) == 0 {
		return "", false, false
	}
	base := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(vals, ", "))
	return base + ";", true, true
}

func parseSelectClauses(head string, rest []string) (query string, autoRun bool, ok bool) {
	var base, fromTable string

	if m := selectAllRe.FindStringSubmatch(head); m != nil {
		fromTable = m[1]
		base = fmt.Sprintf("SELECT * FROM %s", fromTable)
		if m[2] != "" {
			base += " AS " + m[2]
		}
	} else if m := selectColsRe.FindStringSubmatch(head); m != nil {
		cols := strings.Split(m[1], ",")
		for i, c := range cols {
			cols[i] = strings.TrimSpace(c)
		}
		fromTable = m[2]
		base = fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), fromTable)
		if m[3] != "" {
			base += " AS " + m[3]
		}
	} else {
		return "", false, false
	}

	autoRun = true
	var whereConds []string
	for _, clause := range rest {
		clause = strings.TrimSpace(clause)
		if m := joinRe.FindStringSubmatch(clause); m != nil {
			joinKeyword := "JOIN"
			if m[1] == "lj" {
				joinKeyword = "LEFT JOIN"
			}
			joinTable, joinAlias, onLeft, onRight := m[2], m[3], m[4], m[5]
			base += fmt.Sprintf(" %s %s", joinKeyword, joinTable)
			if joinAlias != "" {
				base += " AS " + joinAlias
			}
			if onLeft != "" && onRight != "" {
				base += fmt.Sprintf(" ON %s = %s", onLeft, onRight)
			} else {
				base += fmt.Sprintf(" ON /* TODO: e.g. %s.id=%s.%s_id */ 1=1", fromTable, joinTable, fromTable)
				autoRun = false
			}
			continue
		}
		if m := whereRe.FindStringSubmatch(clause); m != nil {
			whereConds = append(whereConds, m[1])
			continue
		}
		// Not a recognized :j/:lj/:w clause - bail rather than silently
		// dropping something you typed.
		return "", false, false
	}
	if len(whereConds) > 0 {
		base += " WHERE " + strings.Join(whereConds, " AND ")
	}

	return base + ";", autoRun, true
}

var (
	selectAllRe  = regexp.MustCompile(`^sa\s+(\S+?)(?:\s+as\s+(\S+))?$`)
	selectColsRe = regexp.MustCompile(`^s\(([^)]+)\)\s+(\S+?)(?:\s+as\s+(\S+))?$`)
	joinRe       = regexp.MustCompile(`^(lj|j)\s+(\S+?)(?:\s+as\s+(\S+))?(?:\s+on\s+(\S+)=(\S+))?$`)
	whereRe      = regexp.MustCompile(`^w\s+(.+)$`)
	deleteRe     = regexp.MustCompile(`^d\s+(\S+)$`)
	insertRe     = regexp.MustCompile(`^i\s+(\S+)\(([^)]+)\)$`)
)
