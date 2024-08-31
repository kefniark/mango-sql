package internal

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
)

// The parser used has some limitation (based on cockroachDB syntax),
// The preparser is there to normalize the input and avoid a set of known errors.
var alter = []func(string) string{
	removeComments,
	removeTrigger,
	filterValidOperations,
	replaceMysqlUpdate,
	replaceMysqlChartset,
	replacePostgresIndexWhere,
	replacePostgresInherits,
	replacePostgresTypes,
	replaceSubType,
	replaceSQLiteAutoincrement,
	replaceMysqlDateTypes,
	replaceMysqlTextTypes,
	replaceMysqlIntTypes,
	replaceMysqlFloatTypes,
	replaceMysqlDataTypes,
	replaceMysqlBacktips,
	replaceMysqlNumIncrement,
	replaceMysqlKey,
}

func normalize(sql string) string {
	for _, f := range alter {
		sql = f(sql)
	}
	return sql
}

func replaceMysqlBacktips(sql string) string {
	return strings.ReplaceAll(sql, "`", "")
}

var triggerRegexp = regexp.MustCompile(`(?i)(CREATE|ALTER|DROP) (TRIGGER|FUNCTION)[\s\S]*?(END|\$\$|\$_\$)\s*?;`)

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

func replaceSubType(sql string) string {
	return strings.ReplaceAll(sql, "BLOB SUB_TYPE TEXT", "bytea")
}

func replaceMysqlKey(sql string) string {
	regCond := regexp.MustCompile(`(?i)(PRIMARY|FOREIGN|UNIQUE)? KEY .*`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		// fmt.Println("match", sql[match[0]:match[1]], match)
		if match[2] != -1 {
			if !strings.Contains(strings.ToLower(sql[match[2]:match[3]]), "unique") {
				continue
			}
		}
		sql = sql[:match[0]] + " " + sql[match[1]:]
	}

	regCommaCond := regexp.MustCompile(`(?i),\s*?\)`)
	matchesComma := regCommaCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matchesComma)
	for _, match := range matchesComma {
		sql = sql[:match[0]] + ")" + sql[match[1]:]
	}

	return sql
}

func removeComments(sql string) string {
	regCond := regexp.MustCompile(`\/\*.*\*\/;?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " " + sql[match[1]:]
	}

	return sql
}

func replaceMysqlDateTypes(sql string) string {
	regCond := regexp.MustCompile(`(?i)\s(datetime|date|timestamp)(\(\d*\))?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " timestamp" + sql[match[1]:]
	}

	return sql
}

func replaceMysqlIntTypes(sql string) string {
	sql = strings.ReplaceAll(sql, " unsigned ", " ")

	regCond := regexp.MustCompile(`(?i)\s(integer|smallint|tinyint|bigint|int)(\(\d*\))?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " integer" + sql[match[1]:]
	}

	return sql
}

func replaceMysqlFloatTypes(sql string) string {
	regCond := regexp.MustCompile(`(?i)\s(double precision|double|float)(\(\d*\))?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " real" + sql[match[1]:]
	}

	return sql
}

func replaceMysqlTextTypes(sql string) string {
	regCond := regexp.MustCompile(`(?i)\s(mediumtext|longtext|tinytext|nvarchar|character varying|character|char)(\(.*?\))?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " text" + sql[match[1]:]
	}

	regEnums := regexp.MustCompile(`(?i)\s(enum|set)(\(.*?\))`)
	matchesEnums := regEnums.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matchesEnums)
	for _, match := range matchesEnums {
		sql = sql[:match[0]] + " text" + sql[match[1]:]
	}

	return sql
}

func replaceMysqlDataTypes(sql string) string {
	regCond := regexp.MustCompile(`(?i)\s(binary|longblob|mediumblob|tinyblob|blob|tsvector)(\(\d*\))?`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " bytea" + sql[match[1]:]
	}

	return sql
}

func replaceMysqlUpdate(sql string) string {
	regCond := regexp.MustCompile(`(?i)\sON\s(DELETE|UPDATE)\sSET\s\w*`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + " " + sql[match[1]:]
	}

	return sql
}

func replaceMysqlChartset(sql string) string {
	regCond := regexp.MustCompile(`(?i)\s((CHARACTER SET|COLLATE|DEFAULT CHARSET|CHARTSET|ENGINE|AUTO_INCREMENT)[ =]('.*'|.*))[,; ]`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[2]] + " " + sql[match[3]:]
	}

	regComment := regexp.MustCompile(`(?i)\sCOMMENT\s'.*'`)
	matchesComment := regComment.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matchesComment)
	for _, match := range matchesComment {
		sql = sql[:match[0]] + " " + sql[match[1]:]
	}

	return sql
}

func replaceMysqlNumIncrement(sql string) string {
	regCond := regexp.MustCompile(`(?i)(int.*?AUTO_INCREMENT)[,;]`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[2]] + " serial " + sql[match[3]:]
	}

	return sql
}

func replaceSQLiteAutoincrement(sql string) string {
	// for sqlite
	return strings.ReplaceAll(sql, "AUTOINCREMENT", "AUTO_INCREMENT")
}

// Postgres INDEX ON ... WHERE ... is not supported by cockroachDB
func replacePostgresIndexWhere(sql string) string {
	regCond := regexp.MustCompile(`(?i)CREATE INDEX[\s\S]*?(?P<cond>WHERE [\s\S]*?);`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[2]] + sql[match[3]:]
	}

	return sql
}

// Postgres INHERITS keyword is not supported by cockroachDB
func replacePostgresInherits(sql string) string {
	regCond := regexp.MustCompile(`(?i)CREATE TABLE [\s\S]*?(?P<cond>INHERITS[\s\S]*?);`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[2]] + sql[match[3]:]
	}

	return sql
}

func replacePostgresTypes(sql string) string {
	regCond := regexp.MustCompile(`(public\.)?(varchar_pattern_ops|gin_trgm_ops|ts_vector)`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "" + sql[match[1]:]
	}

	return sql
}

type Match struct {
	start int
	end   int
}

func filterValidOperations(sql string) string {
	matches := []Match{}

	for _, reg := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)CREATE TABLE [^;]*?;`),
		regexp.MustCompile(`(?i)ALTER TABLE [^;]*?;`),
		regexp.MustCompile(`(?i)DROP TABLE [^;]*?;`),
		regexp.MustCompile(`(?i)CREATE INDEX [^;]*?;`),
		regexp.MustCompile(`(?i)ALTER INDEX [^;]*?;`),
		regexp.MustCompile(`(?i)DROP INDEX [a-zA-Z.-_]*?;`),
	} {
		res := reg.FindAllStringSubmatchIndex(sql, -1)
		if len(res) == 0 {
			continue
		}

		for _, res := range res {
			txt := sql[res[0]:res[1]]
			if strings.Contains(strings.ToLower(txt), "owner to") {
				continue
			}

			if strings.Contains(strings.ToLower(txt), "to_tsvector") {
				continue
			}

			matches = append(matches, Match{
				start: res[0],
				end:   res[1],
			})
		}
	}

	slices.SortFunc(matches, func(i, j Match) int {
		return cmp.Compare(i.start, j.start)
	})

	chunks := []string{}
	for _, match := range matches {
		chunks = append(chunks, sql[match.start:match.end])
	}

	result := strings.Join(chunks, "\n")
	return result
}
