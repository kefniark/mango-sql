package internal

import (
	"regexp"
	"slices"
	"strings"
)

// The parser used has some limitation (based on cockroachDB syntax),
// The preparser is there to normalize the input and avoid a set of known errors.
var alter = []func(string) string{
	removeTrigger,
	removeCustomTypes,
	replaceSubType,
	replaceDatetime,
	replaceSQLITEAutoincrement,
}

func normalize(sql string) string {
	for _, f := range alter {
		sql = f(sql)
	}
	return sql
}

var triggerRegexp = regexp.MustCompile(`CREATE TRIGGER[\s\S]*?END\s*?;`)

func removeTrigger(sql string) string {
	res := triggerRegexp.FindAllStringSubmatchIndex(sql, -1)
	if len(res) == 0 {
		return sql
	}

	slices.Reverse(res)

	for _, ind := range res {
		sql = sql[:ind[0]] + sql[ind[1]:]
	}

	return sql
}

var customTypeRegexp = regexp.MustCompile(`CREATE TYPE[\s\S]*?;`)

func removeCustomTypes(sql string) string {
	res := customTypeRegexp.FindAllStringSubmatchIndex(sql, -1)
	if len(res) == 0 {
		return sql
	}

	slices.Reverse(res)

	for _, ind := range res {
		sql = sql[:ind[0]] + sql[ind[1]:]
	}

	return sql
}

func replaceSubType(sql string) string {
	return strings.ReplaceAll(sql, "BLOB SUB_TYPE TEXT", "bytea")
}

func replaceDatetime(sql string) string {
	// for mysql & mariadb
	return strings.ReplaceAll(sql, "DATETIME", "TIMESTAMP")
}

func replaceSQLITEAutoincrement(sql string) string {
	// for sqlite
	return strings.ReplaceAll(sql, "AUTOINCREMENT", "AUTO_INCREMENT")
}
