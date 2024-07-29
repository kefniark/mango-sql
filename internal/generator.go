package internal

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
	"golang.org/x/exp/maps"
)

//go:embed templates/*.tmpl
var templates embed.FS

var plural = pluralize.NewClient()

func Generate(schema *SQLSchema, contents io.Writer, pkg string) error {
	headerTmpl, err := template.ParseFS(templates, "templates/header.tmpl")
	if err != nil {
		return err
	}

	factoryTmpl, err := template.ParseFS(templates, "templates/factory.tmpl")
	if err != nil {
		return err
	}

	ctxConstTmpl, err := template.ParseFS(templates, "templates/ctx_const.tmpl")
	if err != nil {
		return err
	}

	ctxTmpl, err := template.ParseFS(templates, "templates/ctx.tmpl")
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

	relationsTmpl, err := template.ParseFS(templates, "templates/relations.tmpl")
	if err != nil {
		return err
	}

	customQueriesTmpl, err := template.ParseFS(templates, "templates/custom.tmpl")
	if err != nil {
		return err
	}

	tables := maps.Values(schema.Tables)
	slices.SortFunc(tables, func(i, j *SQLTable) int {
		return i.Order - j.Order
	})

	postgresTables := []*PostgresTable{}
	for _, table := range tables {
		entry := toPostgresTable(table)
		entry.schema = schema
		entry.table = table
		postgresTables = append(postgresTables, entry)
	}

	postgresQueries := []*PostgresQuery{}
	for _, query := range schema.Queries {
		entry := toPostgresQuery(&query)
		postgresQueries = append(postgresQueries, entry)
	}

	var ctxConstBuf bytes.Buffer
	ctxConst := bufio.NewWriter(&ctxConstBuf)

	for _, table := range postgresTables {
		if err = ctxConstTmpl.Execute(ctxConst, table); err != nil {
			return err
		}
	}
	ctxConst.Flush()

	if err = headerTmpl.Execute(contents, HeaderData{
		Package: pkg,
		Url:     "https://github.com/kefniark/mangosql",
		Date:    time.Now().String(),
		Version: "0.0.1",
	}); err != nil {
		return err
	}

	if err = factoryTmpl.Execute(contents, struct {
		Tables  []*PostgresTable
		Queries []*PostgresQuery
	}{Tables: postgresTables, Queries: postgresQueries}); err != nil {
		return err
	}

	if err = ctxTmpl.Execute(contents, PostgresCtx{
		Const: ctxConstBuf.String(),
	}); err != nil {
		return err
	}

	for _, table := range postgresTables {
		if err = modelTmpl.Execute(contents, table); err != nil {
			return err
		}

		if err = queriesTmpl.Execute(contents, table); err != nil {
			return err
		}
	}

	for _, table := range postgresTables {
		if err = relationsTmpl.Execute(contents, table); err != nil {
			return err
		}
	}

	if err = customQueriesTmpl.Execute(contents, postgresQueries); err != nil {
		return err
	}

	return nil
}

type PostgresCtx struct {
	Const string
}

type PostgresTable struct {
	schema *SQLSchema
	table  *SQLTable

	Name           string
	NameNormalized string
	HasCompositeId bool
	ColumnIds      []*PostgresColumn
	Columns        []*PostgresColumn
	ColumnsCreate  []*PostgresColumn
	ColumnsUpdate  []*PostgresColumn
	Primary        []string
}

type PostgresColumn struct {
	Name           string
	NameNormalized string
	NameJson       string
	Type           string
	TypeSql        string
	Nullable       bool
	HasDefault     bool
}

type PostgresQuery struct {
	Name           string
	NameNormalized string
	Method         string
	Fields         []*PostgresColumn
	SqlQuery       string

	Select         string
	SelectOriginal string
	From           string
	Where          string
	GroupBy        []string
	Having         string
}

func toPostgresQuery(query *SQLQuery) *PostgresQuery {
	fields := []*PostgresColumn{}
	slices.SortFunc(query.SelectFields, func(i, j *SQLColumn) int {
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

func toPostgresTable(table *SQLTable) *PostgresTable {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *SQLColumn) int {
		return i.Order - j.Order
	})

	for _, column := range cols {
		if col := toPostgresColumn(column); col != nil {
			columns = append(columns, col)
		}
	}

	return &PostgresTable{
		Name:           table.Name,
		NameNormalized: strcase.ToCamel(plural.Singular(table.Name)),
		Columns:        columns,
		HasCompositeId: len(getPrimaryFields(table)) > 1,
		ColumnIds:      getIdsFields(table),
		ColumnsCreate:  getCreateFields(table),
		ColumnsUpdate:  getUpdateFields(table),
		Primary:        getPrimaryFields(table),
	}
}

func toPostgresColumn(column *SQLColumn) *PostgresColumn {
	var val string
	if column.Nullable {
		t, ok := postgresNullableType[column.Type]
		if !ok {
			fmt.Println("Unknown type", column.Type)
			return nil
		}
		val = t
	} else {
		t, ok := postgresType[column.Type]
		if !ok {
			fmt.Println("Unknown type", column.Type)
			return nil
		}
		val = t
	}

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
		HasDefault:     column.HasDefault,
	}
}

func getCreateFields(table *SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}
	primary := getPrimaryFields(table)

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *SQLColumn) int {
		return i.Order - j.Order
	})

	for _, column := range cols {
		if slices.Contains(primary, column.Name) {
			continue
		}
		if slices.Contains(getAutogeneratedFields(), column.Name) {
			continue
		}
		columns = append(columns, toPostgresColumn(column))
	}

	return columns
}

func getUpdateFields(table *SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *SQLColumn) int {
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

func getIdsFields(table *SQLTable) []*PostgresColumn {
	columns := []*PostgresColumn{}

	cols := maps.Values(table.Columns)
	slices.SortFunc(cols, func(i, j *SQLColumn) int {
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

func getPrimaryFields(table *SQLTable) []string {
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

var postgresNullableType = map[string]string{
	"int":         "sql.NullInt64",
	"integer":     "sql.NullInt64",
	"float":       "sql.NullFloat64",
	"string":      "sql.NullString",
	"text":        "sql.NullString",
	"varchar":     "sql.NullString",
	"timestamp":   "sql.NullTime",
	"timestamptz": "sql.NullTime",
	"uuid":        "*uuid.UUID",
	"uuid[]":      "[]uuid.UUID",
	"bool":        "bool",
	"string[]":    "[]string",
	"json":        "interface{}",
	"jsonb":       "interface{}",
}
var postgresType = map[string]string{
	"int":         "int64",
	"integer":     "int64",
	"float":       "float64",
	"string":      "string",
	"text":        "string",
	"varchar":     "string",
	"timestamp":   "time.Time",
	"timestamptz": "time.Time",
	"uuid":        "uuid.UUID",
	"uuid[]":      "[]uuid.UUID",
	"bool":        "bool",
	"string[]":    "[]string",
	"json":        "interface{}",
	"jsonb":       "interface{}",
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
	return fmt.Sprintf("%sTable", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetCreateSqlName() string {
	return fmt.Sprintf("%sCreateSql", strcase.ToLowerCamel(plural.Singular(table.Name)))
}

func (table *PostgresTable) GetCreateSqlContent() string {
	fields := []string{}
	keys := []string{}
	values := []string{}

	for _, val := range table.Columns {
		fields = append(fields, val.Name)
	}

	for id, val := range table.ColumnsUpdate {
		keys = append(keys, val.Name)
		values = append(values, fmt.Sprintf("$%d", id+1))
	}

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, "NOW()")
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, "NOW()")
	}

	return fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s`, table.Name, strings.Join(keys, ", "), strings.Join(values, ", "), strings.Join(fields, ", "))
}

func (table *PostgresTable) GetCreateManySqlName() string {
	return fmt.Sprintf("%sCreateManySql", strcase.ToLowerCamel(plural.Singular(table.Name)))
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

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, "NOW()")
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, "NOW()")
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
		values = append(values, "NOW()")
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, "NOW()")
	}

	if table.HasUpdateExtraUpdated() {
		set = append(set, "updated_at=NOW()")
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
	return fmt.Sprintf("%sUpsertManySql", strcase.ToLowerCamel(plural.Singular(table.Name)))
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

	if table.HasInsertExtraCreated() {
		keys = append(keys, "created_at")
		values = append(values, "NOW()")
	}

	if table.HasInsertExtraUpdated() {
		keys = append(keys, "updated_at")
		values = append(values, "NOW()")
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
		values = append(values, "updated_at=NOW()")
	}

	return fmt.Sprintf(`UPDATE %s SET %s WHERE %s RETURNING %s`, table.Name, strings.Join(values, ", "), strings.Join(keys, ", "), strings.Join(fields, ", "))
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

	return fmt.Sprintf(`UPDATE %s SET %s FROM jsonb_to_recordset($1) as t(%s) WHERE %s RETURNING %s`,
		table.Name,
		strings.Join(set, ", "),
		strings.Join(types, ", "),
		strings.Join(where, ", "),
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

	return fmt.Sprintf(`UPDATE %s SET deleted_at=NOW() WHERE %s`, table.Name, strings.Join(keys, ", "))
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

	return fmt.Sprintf(`DELETE FROM %s WHERE %s`, table.Name, strings.Join(keys, ", "))
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

func (table *PostgresTable) GetFieldFilters() []SelectFieldFilter {
	filters := []SelectFieldFilter{}
	for _, col := range table.Columns {
		filters = append(filters, SelectFieldFilter{
			Model: table.NameNormalized,
			Name:  col.NameNormalized,
		})
	}
	return filters
}

func (table *PostgresTable) GetFilters() []SelectFilter {
	filters := []SelectFilter{}
	for _, col := range table.Columns {
		filters = append(filters,
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "Equal",
				Comment: fmt.Sprintf(`Filter to only include %s with a specific %s value`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.Where("%s.%s = $1", arg)`, table.Name, col.Name),
				Args:    []string{fmt.Sprintf("arg %s", col.Type)},
			},
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "NotEqual",
				Comment: fmt.Sprintf(`Filter to exclude %s with a specific %s value`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.Where("%s.%s != $1", arg)`, table.Name, col.Name),
				Args:    []string{fmt.Sprintf("arg %s", col.Type)},
			},
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "In",
				Comment: fmt.Sprintf(`Filter to only include %s with a specific %s value`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.Where("%s.%s = ANY($1)", pq.Array(args))`, table.Name, col.Name),
				Args:    []string{fmt.Sprintf("args ...%s", col.Type)},
			},
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "NotIn",
				Comment: fmt.Sprintf(`Filter to exclude %s with a specific %s value`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.Where("NOT(%s.%s = ANY($1))", pq.Array(args))`, table.Name, col.Name),
				Args:    []string{fmt.Sprintf("args ...%s", col.Type)},
			},
		)

		f := strings.ReplaceAll(col.Type, "*", "")
		isString := slices.Contains([]string{"string"}, f)
		isNumber := slices.Contains([]string{"int64"}, f)
		isDate := slices.Contains([]string{"time.Time", "sql.NullTime"}, f)

		if isNumber {
			filters = append(filters,
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "GreaterThan",
					Comment: fmt.Sprintf(`Filter to only include %s when %s > $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s > $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "GreaterThanOrEqual",
					Comment: fmt.Sprintf(`Filter to only include %s when %s >= $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s >= $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "LesserThan",
					Comment: fmt.Sprintf(`Filter to only include %s when %s > $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s < $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "LesserThanOrEqual",
					Comment: fmt.Sprintf(`Filter to include %s only when %s <= $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s <= $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
			)
		}

		if isDate {
			filters = append(filters,
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "After",
					Comment: fmt.Sprintf(`Filter to only include %s when %s happen after $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s > $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "Before",
					Comment: fmt.Sprintf(`Filter to only include %s when %s happen before $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s < $1", val)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("val %s", col.Type)},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "Between",
					Comment: fmt.Sprintf(`Filter to only include %s when %s happen between $low and $high`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s BETWEEN $1 AND $2", low, high)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("low %s", col.Type), fmt.Sprintf("high %s", col.Type)},
				},
			)
		}

		if isString {
			filters = append(filters,
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "Like",
					Comment: fmt.Sprintf(`Filter to only include %s when %s is LIKE $value`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s ILIKE $1", search)`, table.Name, col.Name),
					Args:    []string{fmt.Sprintf("search %s", col.Type)},
				},
			)
		}

		if col.Nullable {
			filters = append(filters,
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "IsNull",
					Comment: fmt.Sprintf(`Filter to only include %s when %s IS NULL`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s IS NULL")`, table.Name, col.Name),
					Args:    []string{},
				},
				SelectFilter{
					Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
					Name:    "IsNotNull",
					Comment: fmt.Sprintf(`Filter to exclude %s when %s IS NULL`, table.NameNormalized, col.NameNormalized),
					Sql:     fmt.Sprintf(`.Where("%s.%s IS NOT NULL")`, table.Name, col.Name),
					Args:    []string{},
				},
			)
		}

		filters = append(filters,
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "OrderAsc",
				Comment: fmt.Sprintf(`Filter to sort %s by %s is ASC order`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.OrderBy("%s.%s DESC")`, table.Name, col.Name),
				Args:    []string{},
			},
			SelectFilter{
				Model:   fmt.Sprintf("%s%s", table.NameNormalized, col.NameNormalized),
				Name:    "OrderDesc",
				Comment: fmt.Sprintf(`Filter to sort %s by %s is DESC order`, table.NameNormalized, col.NameNormalized),
				Sql:     fmt.Sprintf(`.OrderBy("%s.%s DESC")`, table.Name, col.Name),
				Args:    []string{},
			},
		)
	}

	filters = append(filters,
		SelectFilter{
			Model: table.NameNormalized,
			Name:  "Offset",
			Sql:   `.Offset(val)`,
			Args:  []string{"val uint64"},
		},
		SelectFilter{
			Model: table.NameNormalized,
			Name:  "Limit",
			Sql:   `.Limit(val)`,
			Args:  []string{"val uint64"},
		},
		SelectFilter{
			Model: table.NameNormalized,
			Name:  "Distinct",
			Sql:   `.Distinct()`,
			Args:  []string{},
		},
		SelectFilter{
			Model: table.NameNormalized,
			Name:  "GroupBy",
			Sql:   `.GroupBy(args...)`,
			Args:  []string{"args ...string"},
		},
	)

	return filters
}

type SelectFieldFilter struct {
	Model string
	Name  string
}

type SelectFilter struct {
	Model   string
	Name    string
	Comment string
	Sql     string
	Args    []string
}

type SelectQueryRef struct {
	FromTable  string
	ToTable    string
	Fields     []*PostgresColumn
	GetMethod  string
	Ref        string
	FromAccess string
}

type SelectQuery struct {
	Name   string
	Fields []*PostgresColumn
	Method string
}

type HeaderData struct {
	Package string
	Url     string
	Date    string
	Version string
}
