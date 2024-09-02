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
	replaceMysqlTypes,
	replaceMysqlChartset,
	replacePostgresIndexWhere,
	replacePostgresInherits,
	replacePostgresTypes,
	replacePostgresBinarySubType,
	replaceMysqlBacktips,
	replaceMysqlKey,
}

func normalize(sql string) string {
	for _, f := range alter {
		sql = f(sql)
	}
	return sql
}

func normalizeVarname(original string) string {
	sql := strings.TrimSpace(original)
	sql = strings.Trim(sql, "`")
	sql = strings.Trim(sql, "'")
	sql = strings.Trim(sql, `"`)
	return `"` + sql + `"`
}

var regVarBacktips = regexp.MustCompile(`(?i)\x60\w+\x60`)

func replaceMysqlBacktips(sql string) string {
	matches := regVarBacktips.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + normalizeVarname(sql[match[0]:match[1]]) + sql[match[1]:]
	}

	return sql
}

var regTrigger = regexp.MustCompile(`(?i)(CREATE|ALTER|DROP) (TRIGGER|FUNCTION)[\s\S]*?(END|\$\$|\$_\$)\s*?;`)

func removeTrigger(sql string) string {
	res := regTrigger.FindAllStringSubmatchIndex(sql, -1)
	if len(res) == 0 {
		return sql
	}

	slices.Reverse(res)

	for _, ind := range res {
		sql = sql[:ind[0]] + sql[ind[1]:]
	}

	return sql
}

func replacePostgresBinarySubType(sql string) string {
	return strings.ReplaceAll(sql, "BLOB SUB_TYPE TEXT", "bytea")
}

func replaceMysqlKey(sql string) string {
	regCond := regexp.MustCompile(`(?i)(PRIMARY|FOREIGN|UNIQUE)? KEY .*`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
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

type TableContent struct {
	Name      string
	Content   string
	Start     int
	End       int
	Fields    []TableField
	MetaStart int
	MetaEnd   int
}

type TableField struct {
	Name      string
	Type      string
	Ref       string
	VarStart  int
	VarEnd    int
	TypeStart int
	TypeEnd   int
}

var regFindTable = regexp.MustCompile(`(?i)CREATE TABLE\s+(?P<name>.*?)\s+\((?P<content>[^;]*)\)(?P<meta>.*?)?;`)

func findTableContents(sql string) []TableContent {
	matches := regFindTable.FindAllStringSubmatchIndex(sql, -1)
	res := []TableContent{}
	for _, match := range matches {
		res = append(res, TableContent{
			Name:      sql[match[2]:match[3]],
			Content:   sql[match[4]:match[5]],
			Start:     match[4],
			End:       match[5],
			Fields:    findFields(sql[match[4]:match[5]], match[4]),
			MetaStart: match[6],
			MetaEnd:   match[7],
		})
	}
	return res
}

func findFields(table string, offset int) []TableField {
	quote := 0
	parenthesis := 0
	from := 0
	fields := []TableField{}

	for pos, char := range table {
		if char == '(' {
			parenthesis++
		}
		if char == ')' {
			parenthesis++
		}
		if string(char) == "'" {
			quote++
		}
		if char == '\n' {
			if strings.HasPrefix(strings.TrimSpace(table[from:pos]), "--") {
				from = pos + 1
				continue
			}
		}

		if quote%2 != 0 || parenthesis%2 != 0 {
			continue
		}

		if string(char) == "," || pos == len(table)-1 {
			line := table[from : pos+1]
			if string(char) == "," || string(char) == ";" {
				line = table[from:pos]
			}

			clean := strings.ToLower(strings.TrimSpace(line))
			if strings.HasPrefix(clean, "primary") || strings.HasPrefix(clean, "constraint") || strings.HasPrefix(clean, "unique") || strings.HasPrefix(clean, "foreign") || strings.HasPrefix(clean, "key") {
				from = pos + 1
				continue
			}

			entries := strings.Fields(line)
			varname := strings.TrimSpace(entries[0])
			vartype := strings.TrimSpace(strings.Join(entries[1:], " "))
			start := from + strings.Index(line, varname)

			data := TableField{
				Name:      varname,
				Type:      vartype,
				Ref:       "",
				VarStart:  offset + start,
				VarEnd:    offset + start + len(varname),
				TypeStart: offset + start + len(varname) + 1,
				TypeEnd:   offset + from + len(line),
			}
			data.Type, data.Ref = splitTypeRef(data.Type)

			fields = append(fields, data)
			from = pos + 1
		}
	}

	return fields
}

func splitTypeRef(sql string) (string, string) {
	ref := -1
	if r := strings.Index(strings.ToLower(sql), " references "); r > -1 {
		ref = r
	}
	if r := strings.Index(strings.ToLower(sql), " foreign key "); r > -1 && r < ref {
		ref = r
	}

	if ref > -1 {
		return sql[:ref], sql[ref:]
	}

	return sql, ""
}

func replaceMysqlTypes(sql string) string {
	tables := findTableContents(sql)
	slices.Reverse(tables)
	for _, table := range tables {
		fields := table.Fields
		slices.Reverse(fields)
		for _, field := range fields {
			varName := normalizeVarname(field.Name)
			vartype := field.Type

			vartype = replaceMysqlUpdate(vartype)
			vartype = replaceMysqlIntUnsigned(vartype)
			vartype = replaceMysqlIntTypes(vartype)
			vartype = replaceMysqlFloatTypes(vartype)
			vartype = replaceMysqlTextTypes(vartype)
			vartype = replaceMysqlDataTypes(vartype)
			vartype = replaceMysqlDataSubtypes(vartype)
			vartype = replaceMysqlDateTypes(vartype)
			vartype = replaceMysqlEnumTypes(vartype)
			vartype = replaceMysqlComment(vartype)
			vartype = replaceMysqlNumIncrement(vartype)

			// if field.Type != vartype {
			// 	fmt.Println("Replace", field.Type, "->", vartype)
			// }

			sql = sql[:field.VarStart] + varName + " " + vartype + " " + field.Ref + sql[field.TypeEnd:]
		}
	}

	return sql
}

func replaceMysqlIntUnsigned(sql string) string {
	return strings.ReplaceAll(strings.ReplaceAll(sql, " unsigned ", " "), "UNSIGNED", "")
}

var regLineTypeComment = regexp.MustCompile(`(?i)COMMENT \'.*\'`)

func replaceMysqlComment(sql string) string {
	matches := regLineTypeComment.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "" + sql[match[1]:]
	}

	return sql
}

var regLineTypeDate = regexp.MustCompile(`(?i)(datetime|date|timestamp)(\(\d*\))?`)

func replaceMysqlDateTypes(sql string) string {
	matches := regLineTypeDate.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "timestamp" + sql[match[1]:]
	}

	return sql
}

var regLineTypeInt = regexp.MustCompile(`(?i)(integer|smallint|tinyint|bigint|int)(\([\d, ]*\))?`)

func replaceMysqlIntTypes(sql string) string {
	matches := regLineTypeInt.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "integer" + sql[match[1]:]
	}

	return sql
}

var regLineTypeFloat = regexp.MustCompile(`(?i)(double precision|double|float)(\([\d, ]*\))?`)

func replaceMysqlFloatTypes(sql string) string {
	matches := regLineTypeFloat.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "real" + sql[match[1]:]
	}

	return sql
}

var regLineTypeEnum = regexp.MustCompile(`(?i)(enum|character set|set)(\(.*?\))`)

func replaceMysqlEnumTypes(sql string) string {
	matches := regLineTypeEnum.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "text" + sql[match[1]:]
	}

	return strings.ReplaceAll(sql, "text text", "text")
}

var regLineTypeText = regexp.MustCompile(`(?i)(mediumtext|longtext|tinytext|character set [\w]*|character varying|character|nvarchar|varchar|bpchar|char)(\(.*?\))?`)

func replaceMysqlTextTypes(sql string) string {
	matches := regLineTypeText.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "text" + sql[match[1]:]
	}

	return strings.ReplaceAll(sql, "text text", "text")
}

var regLineTypeBinary = regexp.MustCompile(`(?i)(binary|longblob|mediumblob|tinyblob|blob|tsvector)(\(\d*\))?`)

func replaceMysqlDataTypes(sql string) string {
	matches := regLineTypeBinary.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "bytea" + sql[match[1]:]
	}

	return sql
}

var regLineTypeSubtype = regexp.MustCompile(`(?i)SUB_TYPE \w*`)

func replaceMysqlDataSubtypes(sql string) string {
	matches := regLineTypeSubtype.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[0]] + "" + sql[match[1]:]
	}

	return sql
}

var regLineCascade = regexp.MustCompile(`(?i)\sON\s(DELETE|UPDATE)(\sSET)?\s\w*`)

func replaceMysqlUpdate(sql string) string {
	matches := regLineCascade.FindAllStringSubmatchIndex(sql, -1)
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
	sql = strings.ReplaceAll(sql, "AUTOINCREMENT", "AUTO_INCREMENT")

	regCond := regexp.MustCompile(`(?i)(int.*?AUTO_INCREMENT)`)
	matches := regCond.FindAllStringSubmatchIndex(sql, -1)
	slices.Reverse(matches)
	for _, match := range matches {
		sql = sql[:match[2]] + " serial " + sql[match[3]:]
	}

	return sql
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

var filterOperations = []*regexp.Regexp{
	regexp.MustCompile(`(?i)CREATE TABLE [^;]*?;`),
	regexp.MustCompile(`(?i)ALTER TABLE [^;]*?;`),
	regexp.MustCompile(`(?i)DROP TABLE [^;]*?;`),
	regexp.MustCompile(`(?i)CREATE INDEX [^;]*?;`),
	regexp.MustCompile(`(?i)ALTER INDEX [^;]*?;`),
	regexp.MustCompile(`(?i)DROP INDEX [a-zA-Z.-_]*?;`),
}

func filterValidOperations(sql string) string {
	matches := []Match{}

	for _, reg := range filterOperations {
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

			if strings.Contains(strings.ToLower(txt), "alter column index") {
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
