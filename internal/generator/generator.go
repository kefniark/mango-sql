package generator

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
	"github.com/kefniark/mango-sql/internal/core"
	"golang.org/x/exp/maps"
)

//go:embed templates/*.tmpl
var templates embed.FS

var plural = pluralize.NewClient()

const timeDeps = "time"

var customTypes = []FieldInitializer{
	{
		Import: "github.com/google/uuid",
		Init:   "uuid.New()",
		Type:   "uuid.UUID",
		Match:  "uuid",
	},
}

//nolint:funlen,gocognit,gocyclo,cyclop // Need refactoring
func Generate(schema *core.SQLSchema, contents io.Writer, pkg string, driver string, logger string) error {
	deps := map[string]string{}
	var templateType string
	switch driver {
	case "pgx":
		templateType = "pgx"
	default:
		templateType = "pq"
	}

	if driver == "pq" {
		deps["pq"] = "github.com/lib/pq"
	}

	headerTmpl, err := template.ParseFS(templates, fmt.Sprintf("templates/header_%s.tmpl", templateType))
	if err != nil {
		return err
	}

	factoryTmpl, err := template.ParseFS(templates, fmt.Sprintf("templates/factory_%s.tmpl", templateType))
	if err != nil {
		return err
	}

	loggerTmpl, err := template.ParseFS(templates, fmt.Sprintf("templates/logger_%s.tmpl", logger))
	if err != nil {
		return err
	}

	modelTmpl, err := template.ParseFS(templates, "templates/model.tmpl")
	if err != nil {
		return err
	}

	queriesTmpl, err := template.ParseFS(templates, "templates/queries.tmpl")
	if err != nil {
		return err
	}

	customQueriesTmpl, err := template.ParseFS(templates, "templates/custom.tmpl")
	if err != nil {
		return err
	}

	tables := maps.Values(schema.Tables)
	slices.SortFunc(tables, func(i, j *core.SQLTable) int {
		return i.Order - j.Order
	})

	postgresTables := []*PostgresTable{}
	for _, table := range tables {
		entry := toPostgresTable(table, driver)
		entry.schema = schema
		entry.table = table
		postgresTables = append(postgresTables, entry)
	}

	postgresQueries := []*PostgresQuery{}
	for _, query := range schema.Queries {
		entry := toPostgresQuery(&query)
		postgresQueries = append(postgresQueries, entry)
	}

	for _, t := range postgresTables {
		for _, c := range t.Columns {
			for _, custom := range customTypes {
				if strings.Contains(c.Type, custom.Match+".") {
					deps[custom.Type] = custom.Import
				}
			}
			if strings.HasPrefix(c.Type, "time.") || strings.HasPrefix(c.Type, "[]time.") || strings.HasPrefix(c.Type, "*time.") {
				deps["time"] = timeDeps
			}
			if strings.HasPrefix(c.Type, "sql.") || strings.HasPrefix(c.Type, "[]sql.") || strings.HasPrefix(c.Type, "*sql.") {
				deps["sql"] = "database/sql"
			}
		}
	}

	var ctxConstBuf bytes.Buffer
	ctxConst := bufio.NewWriter(&ctxConstBuf)
	ctxConst.Flush()

	placeholder := "squirrel.Dollar"
	if driver == core.DriverSqlite || driver == core.DriverMysql || driver == core.DriverMariaDB {
		placeholder = "squirrel.Question"
	}

	if driver == core.DriverMysql || driver == core.DriverMariaDB {
		deps["strings"] = "strings"
	}

	logConfig := LoggerConfig{}

	switch logger {
	case "zap":
		deps["zap"] = "go.uber.org/zap"
		deps["time"] = timeDeps
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `*zap.Logger`
	case "logrus":
		deps["logrus"] = "github.com/sirupsen/logrus"
		deps["time"] = timeDeps
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `*logrus.Logger`
	case "zerolog":
		deps["zero"] = "github.com/rs/zerolog"
		deps["time"] = timeDeps
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `zerolog.Logger`
	case "console":
		deps["log"] = "log"
		deps["time"] = timeDeps

		logConfig.HasLogger = true
		logConfig.HasLoggerParam = false
		logConfig.Type = ``
	}

	if err = headerTmpl.Execute(contents, HeaderData{
		Package:     pkg,
		URL:         "https://github.com/kefniark/mangosql",
		Date:        time.Now().String(),
		Version:     "0.0.1",
		Deps:        maps.Values(deps),
		Placeholder: placeholder,
	}); err != nil {
		return err
	}

	_, ok := deps["uuid.UUID"]
	if err = factoryTmpl.Execute(contents, struct {
		Tables  []*PostgresTable
		Queries []*PostgresQuery
		Filters []FilterMethod
		Logger  LoggerConfig
	}{
		Tables:  postgresTables,
		Queries: postgresQueries,
		Filters: GetFilterMethods(postgresTables, driver),
		Logger:  logConfig,
	}); err != nil {
		return err
	}

	if err = loggerTmpl.Execute(contents, nil); err != nil {
		return err
	}

	for _, table := range postgresTables {
		if err = modelTmpl.Execute(contents, struct {
			Table  *PostgresTable
			Logger LoggerConfig
		}{
			Table:  table,
			Logger: logConfig,
		}); err != nil {
			return err
		}

		if err = queriesTmpl.Execute(contents, struct {
			Table   *PostgresTable
			Logger  LoggerConfig
			HasUUID bool
		}{
			Table:   table,
			Logger:  logConfig,
			HasUUID: ok,
		}); err != nil {
			return err
		}
	}

	if err = customQueriesTmpl.Execute(contents, struct {
		Queries []*PostgresQuery
		Logger  LoggerConfig
	}{
		Queries: postgresQueries,
		Logger:  logConfig,
	}); err != nil {
		return err
	}

	return nil
}

func toPostgresQuery(query *core.SQLQuery) *PostgresQuery {
	fields := []*PostgresColumn{}
	slices.SortFunc(query.SelectFields, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})
	for _, field := range query.SelectFields {
		fields = append(fields, toPostgresColumn(field))
	}

	return &PostgresQuery{
		Name:           query.Name,
		NameNormalized: strcase.ToCamel(query.Name),
		Method:         query.Method,
		Fields:         fields,
		SQLQuery:       query.Query,

		Select:         query.Select,
		SelectOriginal: query.SelectOriginal,
		From:           query.From,
		Where:          query.Where,
		GroupBy:        query.GroupBy,
		Having:         query.Having,
	}
}

func toPostgresTable(table *core.SQLTable, driver string) *PostgresTable {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	for _, column := range cols {
		if col := toPostgresColumn(column); col != nil {
			columns = append(columns, col)
		}
	}

	create := getCreateFields(table)
	update := getUpdateFields(table)

	return &PostgresTable{
		driver: driver,

		Name:               table.Name,
		NameNormalized:     strcase.ToCamel(plural.Singular(table.Name)),
		Columns:            columns,
		HasCompositeID:     len(getPrimaryFields(table)) > 1,
		HasIDAutoGenerated: isIDGenerated(table),
		ColumnIDs:          getIDsFields(table),
		ColumnsCreate:      create,
		ColumnsUpdate:      update,
		Primary:            getPrimaryFields(table),
	}
}

func toPostgresColumn(column *core.SQLColumn) *PostgresColumn {
	val := getColumnType(column)

	json := column.As
	if json == "" {
		json = strings.ToLower(strcase.ToSnake(column.Name))
	}

	name := column.Ref
	if name == "" {
		name = column.Name
	}

	return &PostgresColumn{
		Name:           name,
		NameNormalized: strcase.ToCamel(name),
		NameJSON:       json,
		Type:           val,
		TypeSQL:        column.TypeSQL,
		Nullable:       column.Nullable,
		IsArray:        strings.Contains(column.TypeSQL, "[]"),
		HasDefault:     column.HasDefault,
	}
}

func getCreateFields(table *core.SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}
	primary := getPrimaryFields(table)

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	for _, column := range cols {
		if slices.Contains(primary, column.Name) {
			if column.HasDefault {
				continue
			}

			hasCustomType := false
			for _, custom := range customTypes {
				if column.Type == custom.Match {
					hasCustomType = true
					continue
				}
			}

			if hasCustomType {
				continue
			}
		}

		if slices.Contains(getAutogeneratedFields(), column.Name) {
			continue
		}
		columns = append(columns, toPostgresColumn(column))
	}

	return columns
}

func getUpdateFields(table *core.SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	for _, column := range cols {
		if slices.Contains(getAutogeneratedFields(), column.Name) {
			continue
		}
		columns = append(columns, toPostgresColumn(column))
	}

	return columns
}

func isIDGenerated(table *core.SQLTable) bool {
	for _, column := range getIDsFields(table) {
		if column.HasDefault {
			return true
		}
	}
	return false
}

func getIDsFields(table *core.SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	ids := getPrimaryFields(table)
	for _, column := range cols {
		if !slices.Contains(ids, column.Name) {
			continue
		}
		columns = append(columns, toPostgresColumn(column))
	}

	return columns
}

func getPrimaryFields(table *core.SQLTable) []string {
	for _, constraint := range table.Constraints {
		if constraint.Type == "PRIMARY" {
			return constraint.Columns
		}
	}

	if len(table.References) > 0 {
		ids := []string{}
		for _, ref := range table.References {
			ids = append(ids, ref.Columns...)
		}
		return ids
	}

	if len(table.Indexes) > 0 {
		return table.Indexes[0].Columns
	}

	return []string{}
}

func getAutogeneratedFields() []string {
	return []string{"created_at", "updated_at", "deleted_at"}
}

func (table *PostgresTable) HasStructCopy() bool {
	return len(table.ColumnsCreate) == len(table.ColumnsUpdate)
}

func (table *PostgresTable) GetPrimaryKeyConstructors() []PrimaryFieldInit {
	if table.HasCompositeID || table.HasIDAutoGenerated {
		return nil
	}

	fields := []PrimaryFieldInit{}
	for _, f := range table.ColumnIDs {
		for _, custom := range customTypes {
			if f.Type != custom.Type {
				continue
			}

			fields = append(fields, PrimaryFieldInit{
				Name: f.NameNormalized,
				Init: custom.Init,
			})
		}
	}

	return fields
}

func (table *PostgresTable) HasInsertExtraCreated() bool {
	for _, t := range table.Columns {
		if t.NameJSON == "created_at" && !t.HasDefault {
			return true
		}
	}

	return false
}

func (table *PostgresTable) HasInsertExtraUpdated() bool {
	for _, t := range table.Columns {
		if t.NameJSON == "updated_at" && !t.HasDefault {
			return true
		}
	}

	return false
}

func (table *PostgresTable) HasUpdateExtraUpdated() bool {
	for _, t := range table.Columns {
		if t.NameJSON == "updated_at" {
			return true
		}
	}

	return false
}

func (table *PostgresTable) HasInsertReturning() bool {
	return table.driver != core.DriverMysql
}

func (table *PostgresTable) GetCreateSQLContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	columns := table.ColumnsUpdate
	if table.HasIDAutoGenerated {
		columns = table.ColumnsCreate
	}

	for id, val := range columns {
		keys = append(keys, val.Name)
		values = append(values, param(id+1, table.driver))
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, table.GetNow())
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, table.GetNow())
	}

	if table.driver == core.DriverMysql {
		return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s);`, table.Name, strings.Join(keys, ", "), strings.Join(values, ", "))
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s;`, table.Name, strings.Join(keys, ", "), strings.Join(values, ", "), strings.Join(fields, ", "))
}

func (table *PostgresTable) IsInsertMany() bool {
	return table.driver == core.DriverMariaDB || table.driver == core.DriverMysql
}

func (table *PostgresTable) GetCreateManySQLArg() []string {
	if table.driver != core.DriverMysql && table.driver != core.DriverMariaDB {
		return []string{"", "", "0"}
	}

	keys := []string{}
	values := []string{}

	columns := table.ColumnsUpdate
	if table.HasIDAutoGenerated {
		columns = table.ColumnsCreate
	}

	primary := []string{}
	for _, k := range table.GetPrimaryKeyConstructors() {
		primary = append(primary, k.Name)
	}

	for _, val := range columns {
		if !slices.Contains(primary, val.NameNormalized) {
			keys = append(keys, fmt.Sprintf("input.%s", val.NameNormalized))
		}
		values = append(values, "?")
	}

	res := "(" + strings.Join(values, ",") + ")"
	val := `
const values = "` + res + `"
`

	return []string{val, strings.Join(keys, ","), strconv.Itoa(len(keys))}
}

func (table *PostgresTable) GetCreateManySQLContent() string {
	keys := []string{}
	types := []string{}
	ids := []string{}
	values := []string{"*"}

	columns := table.ColumnsUpdate
	if table.HasIDAutoGenerated {
		columns = table.ColumnsCreate
	}

	for _, val := range columns {
		keys = append(keys, val.Name)
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSQL))
		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, fmt.Sprintf("%s.%s", table.Name, val.Name))
		}
	}

	if table.driver == core.DriverSqlite {
		values = nil
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(t.value, '$.%s') %s", val.NameJSON, val.Name))
		}

		if table.HasInsertExtraCreated() {
			keys = append(keys, "created_at")
			values = append(values, table.GetNow())
		}

		if table.HasInsertExtraUpdated() {
			keys = append(keys, "updated_at")
			values = append(values, table.GetNow())
		}

		return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM json_each(?) as t RETURNING %s;`,
			table.Name,
			strings.Join(keys, ", "),
			strings.Join(values, ", "),
			strings.Join(ids, ", "),
		)
	}

	if table.driver == core.DriverMariaDB {
		return fmt.Sprintf(`INSERT INTO %s (%s) VALUES %%s RETURNING %s;`,
			table.Name,
			strings.Join(keys, ", "),
			strings.Join(ids, ", "),
		)
	} else if table.driver == core.DriverMysql {
		return fmt.Sprintf(`INSERT INTO %s (%s) VALUES %%s;`,
			table.Name,
			strings.Join(keys, ", "),
		)
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, table.GetNow())
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, table.GetNow())
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM jsonb_to_recordset($1) as t(%s) RETURNING %s;`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(types, ", "),
		strings.Join(ids, ", "),
	)
}

func (table *PostgresTable) GetUpsertSQLContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}
	ids := []string{}
	set := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, val.Name)
		} else {
			if table.driver == core.DriverMysql || table.driver == core.DriverMariaDB {
				set = append(set, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("VALUES(%s)", val.Name)))
			} else {
				set = append(set, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("EXCLUDED.%s", val.Name)))
			}
		}

		keys = append(keys, val.Name)
		values = append(values, param(id+1, table.driver))
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, table.GetNow())
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, table.GetNow())
	}

	if table.HasUpdateExtraUpdated() {
		set = append(set, "updated_at="+table.GetNow())
	}

	// `INSERT INTO users (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name=VALUES(name);
	if table.driver == core.DriverMysql || table.driver == core.DriverMariaDB {
		return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s;`,
			table.Name,
			strings.Join(keys, ", "),
			strings.Join(values, ", "),
			strings.Join(set, ", "),
		)
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s;`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(ids, ", "),
		strings.Join(set, ", "),
		strings.Join(fields, ", "),
	)
}

func (table *PostgresTable) GetNow() string {
	if table.driver == core.DriverSqlite {
		return "CURRENT_TIMESTAMP"
	}

	return "NOW()"
}

func (table *PostgresTable) GetUpsertManySQLContent() string {
	keys := []string{}
	values := []string{"*"}
	types := []string{}
	ids := []string{}
	returning := []string{}
	set := []string{}

	for _, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, val.Name)
			returning = append(returning, fmt.Sprintf("%s.%s", table.Name, val.Name))
		} else {
			set = append(set, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("EXCLUDED.%s", val.Name)))
		}
		keys = append(keys, val.Name)
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSQL))
	}

	if table.driver == core.DriverSqlite {
		values = nil
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(s.value, '$.%s') AS %s", val.NameJSON, val.Name))
		}

		if table.HasInsertExtraCreated() {
			keys = append(keys, "created_at")
			values = append(values, table.GetNow())
		}

		if table.HasInsertExtraUpdated() {
			keys = append(keys, "updated_at")
			values = append(values, table.GetNow())
		}

		return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM json_each(?) as s WHERE true ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s;`,
			table.Name,
			strings.Join(keys, ", "),
			strings.Join(values, ", "),
			strings.Join(ids, ", "),
			strings.Join(set, ", "),
			strings.Join(returning, ", "),
		)
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, table.GetNow())
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, table.GetNow())
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM jsonb_to_recordset($1) as t(%s) ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s;`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(types, ", "),
		strings.Join(ids, ", "),
		strings.Join(set, ", "),
		strings.Join(returning, ", "),
	)
}

func (table *PostgresTable) HasUpdateReturning() bool {
	return table.driver != core.DriverMysql && table.driver != core.DriverMariaDB
}

func (table *PostgresTable) GetInsertSQLColumnsSorted() []*PostgresColumn {
	return table.ColumnsUpdate
}

func (table *PostgresTable) GetUpdateSQLColumnsSorted() []*PostgresColumn {
	keys := []*PostgresColumn{}
	values := []*PostgresColumn{}

	for _, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, val)
		} else {
			values = append(values, val)
		}
	}

	return append(values, keys...)
}

func (table *PostgresTable) GetUpdateSQLContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	for id, val := range table.GetUpdateSQLColumnsSorted() {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, param(id+1, table.driver)))
		} else {
			values = append(values, fmt.Sprintf("%s=%s", val.Name, param(id+1, table.driver)))
		}
	}

	if table.HasUpdateExtraUpdated() {
		values = append(values, "updated_at="+table.GetNow())
	}

	// sad don't support update returning
	if table.driver == core.DriverMysql || table.driver == core.DriverMariaDB {
		return fmt.Sprintf(`UPDATE %s SET %s WHERE %s;`, table.Name, strings.Join(values, ", "), strings.Join(keys, " AND "))
	}

	return fmt.Sprintf(`UPDATE %s SET %s WHERE %s RETURNING %s;`, table.Name, strings.Join(values, ", "), strings.Join(keys, " AND "), strings.Join(fields, ", "))
}

func (table *PostgresTable) GetUpdateManySQLContent() string {
	ids := []string{}
	set := []string{}
	types := []string{}
	where := []string{}

	for _, val := range table.ColumnsUpdate {
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSQL))

		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, fmt.Sprintf("%s.%s", table.Name, val.Name))
			where = append(where, fmt.Sprintf("%s.%s=t.%s", table.Name, val.Name, val.Name))
		} else {
			set = append(set, fmt.Sprintf("%s=t.%s", val.Name, val.Name))
		}
	}

	if table.driver == core.DriverSqlite {
		values := []string{}
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(s.value, '$.%s') AS %s", val.NameJSON, val.Name))
		}

		return fmt.Sprintf(`UPDATE %s SET %s FROM (SELECT %s FROM json_each(?) as s) as t WHERE %s RETURNING %s;`,
			table.Name,
			strings.Join(set, ", "),
			strings.Join(values, ", "),
			strings.Join(where, " AND "),
			strings.Join(ids, ", "),
		)
	}

	return fmt.Sprintf(`UPDATE %s SET %s FROM jsonb_to_recordset($1) as t(%s) WHERE %s RETURNING %s;`,
		table.Name,
		strings.Join(set, ", "),
		strings.Join(types, ", "),
		strings.Join(where, " AND "),
		strings.Join(ids, ", "),
	)
}

func (table *PostgresTable) GetDeleteSoftSQLName() string {
	fields := []string{}
	for _, column := range table.Columns {
		fields = append(fields, column.Name)
	}

	if !slices.Contains(fields, "deleted_at") {
		return ""
	}

	return fmt.Sprintf("%sDeleteSoftSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetDeleteSoftSQLContent() string {
	keys := []string{}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, param(id+1, table.driver)))
		}
	}

	return fmt.Sprintf(`UPDATE %s SET deleted_at=%s WHERE %s;`, table.Name, table.GetNow(), strings.Join(keys, " AND "))
}

func (table *PostgresTable) GetDeleteHardSQLContent() string {
	keys := []string{}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, param(id+1, table.driver)))
		}
	}

	return fmt.Sprintf(`DELETE FROM %s WHERE %s;`, table.Name, strings.Join(keys, " AND "))
}

func (table *PostgresTable) GetSelectPrimarySQL() []SelectQuery {
	entries := []SelectQuery{}

	entries = append(entries, SelectQuery{
		Name:   table.NameNormalized,
		Method: "FindById",
		Fields: table.ColumnIDs,
	})

	return entries
}

func param(id int, driver string) string {
	if driver == core.DriverMysql || driver == core.DriverMariaDB {
		return "?"
	}
	return fmt.Sprintf("$%d", id)
}

type FieldInitializer struct {
	Import string
	Init   string
	Type   string
	Match  string
}

type PrimaryFieldInit struct {
	Name   string
	Init   string
	Setter string
}

type LoggerConfig struct {
	HasLoggerParam bool
	HasLogger      bool
	Type           string
}
