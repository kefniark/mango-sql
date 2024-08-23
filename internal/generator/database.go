package generator

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"slices"
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

var customTypes = []FieldInitializer{
	{
		Import: "github.com/google/uuid",
		Init:   "uuid.New()",
		Type:   "uuid.UUID",
		Match:  "uuid",
	},
}

func Generate(schema *core.SQLSchema, contents io.Writer, pkg string, driver string, logger string) error {
	fmt.Println(schema)

	templateType := ""
	switch driver {
	case "pgx":
		templateType = "pgx"
	default:
		templateType = "pq"
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

	deps := map[string]string{}
	for _, t := range postgresTables {
		for _, c := range t.Columns {
			for _, custom := range customTypes {
				if strings.Contains(c.Type, custom.Match+".") {
					deps[custom.Type] = custom.Import
				}
			}
			if strings.HasPrefix(c.Type, "time.") || strings.HasPrefix(c.Type, "[]time.") || strings.HasPrefix(c.Type, "*time.") {
				deps["time"] = "time"
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
	if driver == "sqlite" {
		placeholder = "squirrel.Question"
	}

	logConfig := LoggerConfig{}

	switch logger {
	case "zap":
		deps["zap"] = "go.uber.org/zap"
		deps["time"] = "time"
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `*zap.Logger`
	case "logrus":
		deps["logrus"] = "github.com/sirupsen/logrus"
		deps["time"] = "time"
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `*logrus.Logger`
	case "zerolog":
		deps["zero"] = "github.com/rs/zerolog"
		deps["time"] = "time"
		logConfig.HasLogger = true
		logConfig.HasLoggerParam = true
		logConfig.Type = `zerolog.Logger`
	case "console":
		deps["log"] = "log"
		deps["time"] = "time"

		logConfig.HasLogger = true
		logConfig.HasLoggerParam = false
		logConfig.Type = ``
	}

	fmt.Println(logger, logConfig)

	if err = headerTmpl.Execute(contents, HeaderData{
		Package:     pkg,
		Url:         "https://github.com/kefniark/mangosql",
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
		SqlQuery:       query.Query,

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
		HasCompositeId:     len(getPrimaryFields(table)) > 1,
		HasIdAutoGenerated: isIdGenerated(table),
		ColumnIds:          getIdsFields(table),
		ColumnsCreate:      create,
		ColumnsUpdate:      update,
		Primary:            getPrimaryFields(table),
	}
}

func toPostgresColumn(column *core.SQLColumn) *PostgresColumn {
	val := getColumnType(column)
	fmt.Println(column, val)

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
		NameJson:       json,
		Type:           val,
		TypeSql:        column.TypeSql,
		Nullable:       column.Nullable,
		IsArray:        strings.Contains(column.TypeSql, "[]"),
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

func isIdGenerated(table *core.SQLTable) bool {
	for _, column := range getIdsFields(table) {
		if column.HasDefault {
			return true
		}
	}
	return false
}

func getIdsFields(table *core.SQLTable) []*PostgresColumn {
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

	cols := []string{}
	for _, col := range table.Columns {
		cols = append(cols, col.Name)
	}

	return []string{cols[0]}
}

func getAutogeneratedFields() []string {
	return []string{"created_at", "updated_at", "deleted_at"}
}

func (table *PostgresTable) HasStructCopy() bool {
	return len(table.ColumnsCreate) == len(table.ColumnsUpdate)
}

func (table *PostgresTable) GetPrimaryKeyConstructors() []PrimaryFieldInit {
	if table.HasCompositeId || table.HasIdAutoGenerated {
		return nil
	}

	fields := []PrimaryFieldInit{}
	for _, f := range table.ColumnIds {
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
		if t.NameJson == "created_at" && !t.HasDefault {
			return true
		}
	}

	return false
}

func (table *PostgresTable) HasInsertExtraUpdated() bool {
	for _, t := range table.Columns {
		if t.NameJson == "updated_at" && !t.HasDefault {
			return true
		}
	}

	return false
}

func (table *PostgresTable) HasUpdateExtraUpdated() bool {
	for _, t := range table.Columns {
		if t.NameJson == "updated_at" {
			return true
		}
	}

	return false
}

func (table *PostgresTable) GetTableName() string {
	return strcase.ToLowerCamel(plural.Singular(table.Name)) + "Table"
}

func (table *PostgresTable) GetCreateSqlName() string {
	return strcase.ToLowerCamel(plural.Singular(table.Name)) + "CreateSql"
}

func (table *PostgresTable) GetCreateSqlContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	columns := table.ColumnsUpdate
	if table.HasIdAutoGenerated {
		columns = table.ColumnsCreate
	}

	for id, val := range columns {
		keys = append(keys, val.Name)
		values = append(values, fmt.Sprintf("$%d", id+1))
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, table.GetNow())
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, table.GetNow())
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s`, table.Name, strings.Join(keys, ", "), strings.Join(values, ", "), strings.Join(fields, ", "))
}

func (table *PostgresTable) GetCreateManySqlName() string {
	return strcase.ToLowerCamel(plural.Singular(table.Name)) + "CreateManySql"
}

func (table *PostgresTable) GetCreateManySqlContent() string {
	keys := []string{}
	types := []string{}
	ids := []string{}
	values := []string{"*"}

	for _, val := range table.ColumnsUpdate {
		keys = append(keys, val.Name)
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSql))
		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, fmt.Sprintf("%s.%s", table.Name, val.Name))
		}
	}

	if table.driver == "sqlite" {
		values = nil
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(t.value, '$.%s') %s", val.NameJson, val.Name))
		}

		if table.HasInsertExtraCreated() {
			keys = append(keys, "created_at")
			values = append(values, table.GetNow())
		}

		if table.HasInsertExtraUpdated() {
			keys = append(keys, "updated_at")
			values = append(values, table.GetNow())
		}

		return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM json_each(?) as t RETURNING %s`,
			table.Name,
			strings.Join(keys, ", "),
			strings.Join(values, ", "),
			strings.Join(ids, ", "),
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

	return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM jsonb_to_recordset($1) as t(%s) RETURNING %s`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(types, ", "),
		strings.Join(ids, ", "),
	)
}

func (table *PostgresTable) GetUpsertSqlName() string {
	return fmt.Sprintf("%sUpsertSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetUpsertSqlContent() string {
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
			set = append(set, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("EXCLUDED.%s", val.Name)))
		}

		keys = append(keys, val.Name)
		values = append(values, fmt.Sprintf("$%d", id+1))
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

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(ids, ", "),
		strings.Join(set, ", "),
		strings.Join(fields, ", "),
	)
}

func (table *PostgresTable) GetUpsertManySqlName() string {
	return strcase.ToLowerCamel(plural.Singular(table.Name)) + "UpsertManySql"
}

func (table *PostgresTable) GetNow() string {
	if table.driver == "sqlite" {
		return "CURRENT_TIMESTAMP"
	}

	return "NOW()"
}

func (table *PostgresTable) GetUpsertManySqlContent() string {
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
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSql))
	}

	if table.driver == "sqlite" {
		values = nil
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(s.value, '$.%s') AS %s", val.NameJson, val.Name))
		}

		if table.HasInsertExtraCreated() {
			keys = append(keys, "created_at")
			values = append(values, table.GetNow())
		}

		if table.HasInsertExtraUpdated() {
			keys = append(keys, "updated_at")
			values = append(values, table.GetNow())
		}

		return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM json_each(?) as s WHERE true ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s`,
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

	return fmt.Sprintf(`INSERT INTO %s (%s) SELECT %s FROM jsonb_to_recordset($1) as t(%s) ON CONFLICT(%s) DO UPDATE SET %s RETURNING %s`,
		table.Name,
		strings.Join(keys, ", "),
		strings.Join(values, ", "),
		strings.Join(types, ", "),
		strings.Join(ids, ", "),
		strings.Join(set, ", "),
		strings.Join(returning, ", "),
	)
}

func (table *PostgresTable) GetUpdateSqlName() string {
	return fmt.Sprintf("%sUpdateSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetUpdateSqlContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("$%d", id+1)))
		} else {
			values = append(values, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("$%d", id+1)))
		}
	}

	if table.HasUpdateExtraUpdated() {
		values = append(values, "updated_at="+table.GetNow())
	}

	return fmt.Sprintf(`UPDATE %s SET %s WHERE %s RETURNING %s`, table.Name, strings.Join(values, ", "), strings.Join(keys, " AND "), strings.Join(fields, ", "))
}

func (table *PostgresTable) GetUpdateManySqlName() string {
	return fmt.Sprintf("%sUpdateManySql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetUpdateManySqlContent() string {
	ids := []string{}
	set := []string{}
	types := []string{}
	where := []string{}

	for _, val := range table.ColumnsUpdate {
		types = append(types, fmt.Sprintf("%s %s", val.Name, val.TypeSql))

		if slices.Contains(table.Primary, val.Name) {
			ids = append(ids, fmt.Sprintf("%s.%s", table.Name, val.Name))
			where = append(where, fmt.Sprintf("%s.%s=t.%s", table.Name, val.Name, val.Name))
		} else {
			set = append(set, fmt.Sprintf("%s=t.%s", val.Name, val.Name))
		}
	}

	if table.driver == "sqlite" {
		values := []string{}
		for _, val := range table.ColumnsUpdate {
			values = append(values, fmt.Sprintf("json_extract(s.value, '$.%s') AS %s", val.NameJson, val.Name))
		}

		return fmt.Sprintf(`UPDATE %s SET %s FROM (SELECT %s FROM json_each(?) as s) as t WHERE %s RETURNING %s`,
			table.Name,
			strings.Join(set, ", "),
			strings.Join(values, ", "),
			strings.Join(where, " AND "),
			strings.Join(ids, ", "),
		)
	}

	return fmt.Sprintf(`UPDATE %s SET %s FROM jsonb_to_recordset($1) as t(%s) WHERE %s RETURNING %s`,
		table.Name,
		strings.Join(set, ", "),
		strings.Join(types, ", "),
		strings.Join(where, " AND "),
		strings.Join(ids, ", "),
	)
}

func (table *PostgresTable) GetDeleteSoftSqlName() string {
	fields := []string{}
	for _, column := range table.Columns {
		fields = append(fields, column.Name)
	}

	if !slices.Contains(fields, "deleted_at") {
		return ""
	}

	return fmt.Sprintf("%sDeleteSoftSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetDeleteSoftSqlContent() string {
	keys := []string{}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("$%d", id+1)))
		}
	}

	return fmt.Sprintf(`UPDATE %s SET deleted_at=%s WHERE %s`, table.Name, table.GetNow(), strings.Join(keys, " AND "))
}

func (table *PostgresTable) GetDeleteHardSqlName() string {
	return fmt.Sprintf("%sDeleteHardSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetDeleteHardSqlContent() string {
	keys := []string{}

	for id, val := range table.ColumnsUpdate {
		if slices.Contains(table.Primary, val.Name) {
			keys = append(keys, fmt.Sprintf("%s=%s", val.Name, fmt.Sprintf("$%d", id+1)))
		}
	}

	return fmt.Sprintf(`DELETE FROM %s WHERE %s`, table.Name, strings.Join(keys, " AND "))
}

func (table *PostgresTable) GetSelectPrimarySql() []SelectQuery {
	entries := []SelectQuery{}

	entries = append(entries, SelectQuery{
		Name:   table.NameNormalized,
		Method: "FindById",
		Fields: table.ColumnIds,
	})

	return entries
}

func (table *PostgresTable) GetSelectSql() []SelectQueryRef {
	entries := []SelectQueryRef{}

	for _, ref := range table.table.References {
		refFields := []*PostgresColumn{}
		for _, column := range table.Columns {
			if !slices.Contains(ref.Columns, column.Name) {
				continue
			}
			refFields = append(refFields, column)
		}

		entries = append(entries, SelectQueryRef{
			FromTable:  table.NameNormalized,
			ToTable:    strcase.ToCamel(plural.Singular(ref.Table)),
			Ref:        strcase.ToCamel(plural.Singular(ref.Table)),
			GetMethod:  fmt.Sprintf("GetBy%s", strcase.ToCamel(strings.Join(ref.Columns, "_"))),
			FromAccess: strcase.ToCamel(plural.Plural(table.Name)),
			Fields:     refFields,
		})
	}

	return entries
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
