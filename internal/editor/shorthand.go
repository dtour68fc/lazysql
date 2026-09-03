package editor

import (
	"fmt"
	"regexp"
	"strings"
)

// parseShorthandQuery turns the vim-shorthand DSL typed in command mode
// into real SQL. Grammar (space-separated, one leading ":" already
// stripped by vimtea before we ever see it):
//
//	sa <table>                        -> SELECT * FROM <table>;
//	s(col1,col2,...) <table>          -> SELECT col1, col2 FROM <table>;
//	<either of the above> :j <table>  -> ... JOIN <table>
//	                        on <a.col>=<b.col>
//	                                   -> ... JOIN <table> ON <a.col> = <b.col>
//
// Multiple " :j ..." clauses can chain for multi-way joins. If a :j clause
// doesn't give an explicit "on" condition, we don't guess a foreign key
// (guessing wrong silently is worse than not guessing at all) - instead we
// seed a TODO placeholder and tell the caller not to auto-run it, so you
// can fill in the real condition before running it yourself.
//
// autoRun is false whenever any join is missing its "on" condition -
// running a JOIN with no condition on a real database is either a syntax
// error or (worse) an accidental cross join, so we'd rather hand you a
// buffer to finish than fire that off automatically.
func parseShorthandQuery(raw string) (query string, autoRun bool, ok bool) {
	clauses := strings.Split(raw, " :")
	if len(clauses) == 0 {
		return "", false, false
	}

	head := strings.TrimSpace(clauses[0])
	var base string
	var fromTable string

	if m := selectAllRe.FindStringSubmatch(head); m != nil {
		fromTable = m[1]
		base = fmt.Sprintf("SELECT * FROM %s", fromTable)
	} else if m := selectColsRe.FindStringSubmatch(head); m != nil {
		cols := strings.Split(m[1], ",")
		for i, c := range cols {
			cols[i] = strings.TrimSpace(c)
		}
		fromTable = m[2]
		base = fmt.Sprintf("SELECT %s FROM %s", strings.Join(cols, ", "), fromTable)
	} else {
		return "", false, false
	}

	autoRun = true
	for _, clause := range clauses[1:] {
		clause = strings.TrimSpace(clause)
		m := joinRe.FindStringSubmatch(clause)
		if m == nil {
			// Not a recognized :j clause - bail rather than silently drop it.
			return "", false, false
		}
		joinTable, onLeft, onRight := m[1], m[2], m[3]
		if onLeft != "" && onRight != "" {
			base += fmt.Sprintf(" JOIN %s ON %s = %s", joinTable, onLeft, onRight)
		} else {
			base += fmt.Sprintf(" JOIN %s ON /* TODO: e.g. %s.id=%s.%s_id */ 1=1", joinTable, fromTable, joinTable, fromTable)
			autoRun = false
		}
	}

	return base + ";", autoRun, true
}

var (
	selectAllRe  = regexp.MustCompile(`^sa\s+(\S+)$`)
	selectColsRe = regexp.MustCompile(`^s\(([^)]+)\)\s+(\S+)$`)
	joinRe       = regexp.MustCompile(`^j\s+(\S+?)(?:\s+on\s+(\S+)=(\S+))?$`)
)
