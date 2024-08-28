package internal

import (
	"regexp"
	"slices"
	"strings"
)

// The parser used has some limitation (based on cockroachDB syntax),
// The preparser is there to normalize the input and avoid a set of known errors.
var alter = []func(string) string{
	removeInit,
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

func removeInit(sql string) string {
	// var regInitSet = regexp.MustCompile(`SET[\s\S]*?;`)
	var regInitOwner = regexp.MustCompile(`ALTER TABLE [\s\S]*? OWNER TO \S*?;`)
	var regInitSequence = regexp.MustCompile(`CREATE SEQUENCE [\s\S]*?;`)
	var regInitSequenceAlter = regexp.MustCompile(`ALTER SEQUENCE [\s\S]*?;`)
	var regInitExt = regexp.MustCompile(`CREATE EXTENSION [\s\S]*?;`)
	var regInitExtComment = regexp.MustCompile(`COMMENT ON EXTENSION [\s\S]*?;`)

	for _, reg := range []*regexp.Regexp{
		// regInitSet,
		regInitOwner,
		regInitSequence,
		regInitSequenceAlter,
		regInitExt,
		regInitExtComment,
	} {
		res := reg.FindAllStringSubmatchIndex(sql, -1)
		if len(res) == 0 {
			return sql
		}

		slices.Reverse(res)

		for _, ind := range res {
			sql = sql[:ind[0]] + sql[ind[1]:]
		}
	}

	return sql
}

func removeCustomTypes(sql string) string {
	var regTypeCreate = regexp.MustCompile(`CREATE TYPE[\s\S]*?;`)
	var regTypeAlter = regexp.MustCompile(`ALTER TYPE[\s\S]*?;`)
	var regDomainCreate = regexp.MustCompile(`CREATE DOMAIN[\s\S]*?;`)
	var regDomainAlter = regexp.MustCompile(`ALTER DOMAIN[\s\S]*?;`)
	var regFuncCreate = regexp.MustCompile(`CREATE FUNCTION[\s\S]*?;`)
	var regFuncAlter = regexp.MustCompile(`ALTER FUNCTION[\s\S]*?;`)

	for _, reg := range []*regexp.Regexp{
		regTypeCreate,
		regTypeAlter,
		regDomainCreate,
		regDomainAlter,
		regFuncCreate,
		regFuncAlter,
	} {
		res := reg.FindAllStringSubmatchIndex(sql, -1)
		if len(res) == 0 {
			return sql
		}

		slices.Reverse(res)

		for _, ind := range res {
			sql = sql[:ind[0]] + sql[ind[1]:]
		}
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
