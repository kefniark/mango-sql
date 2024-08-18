package internal

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
	"github.com/iancoleman/strcase"
	"github.com/kefniark/mango-sql/internal/core"
)

func ParseQueries(schema *core.SQLSchema, sql string) error {
	stmts, err := parser.Parse(sql)
	if err != nil {
		return fmt.Errorf("schema parsing error: %w", err)
	}

	macro := findQueriesMacro(sql)

	w := &walk.AstWalker{
		Fn: func(ctx interface{}, node interface{}) (stop bool) {
			switch stmt := node.(type) {
			case *tree.SelectClause:
				parseSelectQuery(schema, stmt, macro)
				return true
			}
			return false
		},
	}

	_, err = w.Walk(stmts, nil)
	return err
}

var parseMacro = regexp.MustCompile(`(?m)-- (?P<method>.*):(?P<name>.*)(?P<sql>[^;]*);`)

func findQueriesMacro(sql string) []core.QueryMacro {
	macro := []core.QueryMacro{}
	for _, res := range parseMacro.FindAllStringSubmatch(sql, -1) {
		macro = append(macro, core.QueryMacro{
			Method: res[1],
			Name:   res[2],
			Query:  normalizeSql(res[3]),
		})
	}
	return macro
}

func normalizeSql(sql string) string {
	return strings.ToLower(strings.Join(strings.Fields(sql), ""))
}

func ParseSchema(sql string) (*core.SQLSchema, error) {
	sql = normalize(sql)
	stmts, err := parser.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("schema parsing error: %w", err)
	}

	schema := &core.SQLSchema{
		Tables: make(map[string]*core.SQLTable),
	}

	w := &walk.AstWalker{
		Fn: func(ctx interface{}, node interface{}) (stop bool) {
			switch stmt := node.(type) {
			case *tree.CreateTable:
				parseTable(schema, stmt)
			case *tree.CreateIndex:
				refTable := schema.Tables[stmt.Table.TableName.Normalize()]
				if stmt.Unique {
					refTable.Constraints = append(refTable.Constraints, &core.SQLTableConstraint{
						Name:    stmt.Name.Normalize(),
						Columns: NamesToStrings(stmt.Columns),
						Type:    "UNIQUE",
					})
				}
				refTable.Indexes = append(refTable.Indexes, &core.SQLTableIndex{
					Name:    stmt.Name.Normalize(),
					Columns: NamesToStrings(stmt.Columns),
				})
			case *tree.DropTable:
				for _, table := range stmt.Names {
					delete(schema.Tables, table.TableName.Normalize())
				}
			case *tree.RenameTable:
				from := stmt.Name.ToTableName().TableName.Normalize()
				target := stmt.NewName.ToTableName().TableName.Normalize()
				schema.Tables[target] = schema.Tables[from]
				delete(schema.Tables, from)

			case *tree.AlterTable:
				parseAlterTable(schema, stmt)
			}
			return false
		},
	}

	_, err = w.Walk(stmts, nil)

	for _, table := range schema.Tables {
		for _, ref := range table.References {
			refTable := schema.Tables[ref.Table]

			refTable.Referenced = append(refTable.Referenced, &core.SQLTableReference{
				Name:         ref.Name,
				Columns:      ref.TableColumns,
				Table:        table.Name,
				TableColumns: ref.Columns,
			})
		}
	}

	return schema, err
}

func NameListToStrings(names tree.NameList) []string {
	columns := []string{}
	for _, name := range names {
		columns = append(columns, name.Normalize())
	}
	return columns
}

func NamesToStrings(names tree.IndexElemList) []string {
	columns := []string{}
	for _, name := range names {
		columns = append(columns, name.Column.Normalize())
	}
	return columns
}

func parseSelectQuery(schema *core.SQLSchema, stmt *tree.SelectClause, macro []core.QueryMacro) {
	query := findTableDeps(schema, stmt, macro)
	if query == nil {
		return
	}

	schema.Queries = append(schema.Queries, *query)
}

func findQueryMacro(query string, macro []core.QueryMacro) *core.QueryMacro {
	for _, m := range macro {
		if strings.HasPrefix(normalizeSql(query), m.Query) {
			return &m
		}
	}
	return nil
}

func findTableDeps(schema *core.SQLSchema, table *tree.SelectClause, macro []core.QueryMacro) *core.SQLQuery {
	query := core.SQLQuery{
		Query: table.String(),
	}

	m := findQueryMacro(query.Query, macro)
	if m == nil {
		return nil
	}
	query.Method = strings.TrimSpace(m.Method)
	query.Name = strings.TrimSpace(m.Name)

	var walk func(ctx *FindTableDepsCtx, nodes ...interface{})

	walk = func(ctx *FindTableDepsCtx, nodes ...interface{}) {
		for _, n := range nodes {
			switch s := n.(type) {
			case tree.TableExprs:
				for _, expr := range s {
					walk(ctx, expr)
				}
			case *tree.Subquery:
			case *tree.ParenSelect:
				walk(ctx, s.Select)
			case *tree.Select:
				walk(ctx, s.Select)
			case *tree.TableName:
				ctx.tables = append(ctx.tables, s.TableName.Normalize())
				// fmt.Println("TableName", s.TableName)
				query.From = s.String()
			case *tree.AliasedTableExpr:
				subCtx := &FindTableDepsCtx{}
				walk(subCtx, s.Expr)
				ctx.tables = append(ctx.tables, subCtx.tables...)
				ctx.Tables = append(ctx.Tables, core.TableDeps{
					Names: subCtx.tables,
					As:    s.As.Alias.Normalize(),
				})
				// fmt.Println("AliasedTableExpr", s)
			case *tree.JoinTableExpr:
				walk(ctx, s.Left)
				walk(ctx, s.Right)
				query.From = s.String()
				// fmt.Println("JoinTableExpr", s)
			case *tree.Where:
				if s != nil && s.Expr != nil {
					query.Where = s.Expr.String()
				}
			default:
				fmt.Printf("not supported : %T %s\n", s, s)
			}
		}
	}

	ctx := &FindTableDepsCtx{}
	walk(ctx, table.From.Tables)
	walk(ctx, table.Where)

	// group by
	for _, e := range table.GroupBy {
		query.GroupBy = append(query.GroupBy, e.String())
	}

	// having
	if table.Having != nil {
		query.Having = table.Having.Expr.String()
	}

	names := []string{}
	for i, st := range table.Exprs {
		field := st.Expr.String()
		asClean := strings.TrimSpace(strings.Trim(st.As.String(), `"'`))

		cols := resolveTableColumns(field, asClean, ctx.Tables, schema, i)
		query.SelectFields = append(query.SelectFields, cols...)
		names = append(names, field)
	}

	query.Select = strings.Join(names, ", ")
	query.SelectOriginal = query.Select
	query.SelectFields = slices.Compact(query.SelectFields)
	slices.SortFunc(query.SelectFields, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	selectInput := []string{}
	for _, sel := range query.SelectFields {
		selectInput = append(selectInput, fmt.Sprintf("%s AS %s", sel.Name, sel.As))
	}
	query.Select = strings.Join(selectInput, ", ")

	return &query
}

func resolveTableColumns(field string, as string, tables []core.TableDeps, schema *core.SQLSchema, i int) []*core.SQLColumn {
	prefix := ""
	name := strings.TrimSpace(field)
	columns := []*core.SQLColumn{}

	if strings.Contains(field, ".") {
		split := strings.Split(field, ".")
		if !strings.Contains(split[0], "(") {
			prefix = strings.TrimSpace(split[0])
			name = strings.TrimSpace(split[1])
		}
	}

	for j, table := range tables {
		_, ok := schema.Tables[prefix]
		if prefix != "" && prefix != table.As && !ok {
			continue
		}

		for k, t := range table.Names {
			c, ok := schema.Tables[t]
			if !ok {
				continue
			}

			if t != c.Name {
				continue
			}

			if prefix != "" && prefix != table.As && prefix != c.Name {
				continue
			}

			tableName := t
			if table.As != "" {
				tableName = table.As
			}

			if name == "*" {
				for _, column := range c.Columns {
					name := fmt.Sprintf("%s.%s", tableName, column.Name)
					as := fmt.Sprintf("%s.%d", column.Name, j)
					if j == 0 {
						as = column.Name
					}
					columns = append(columns, &core.SQLColumn{
						Table:    t,
						TableAs:  tableName,
						Ref:      strcase.ToCamel(fmt.Sprintf("%s.%s", tableName, column.Name)),
						Name:     name,
						As:       strings.ToLower(strcase.ToSnake(fmt.Sprintf("%s_%s", tableName, as))),
						Type:     column.Type,
						TypeSql:  column.TypeSql,
						Nullable: column.Nullable,
						Order:    i*1000000 + j*10000 + k*100 + column.Order,
					})
				}
			} else if field, ok := c.Columns[name]; ok {
				name := fmt.Sprintf("%s.%s", tableName, field.Name)
				as := fmt.Sprintf("%s.%d", field.Name, j)
				if j == 0 {
					as = field.Name
				}

				columns = append(columns, &core.SQLColumn{
					Table:    name,
					TableAs:  tableName,
					Ref:      strcase.ToCamel(fmt.Sprintf("%s.%s", tableName, field.Name)),
					Name:     name,
					As:       strings.ToLower(strcase.ToSnake(fmt.Sprintf("%s_%s", tableName, as))),
					Type:     field.Type,
					TypeSql:  field.TypeSql,
					Nullable: field.Nullable,
					Order:    i*1000000 + j*10000 + k*100 + field.Order,
				})
			}
		}
	}

	if len(columns) == 0 {
		cleanupName := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(field, "(", "."), ")", "."), "-.")

		fieldType := "string"
		fieldSqlType := "UNKNOWN"

		for _, n := range []string{"count("} {
			if strings.HasPrefix(strings.ToLower(name), n) {
				fieldType = "int"
			}
		}
		for _, n := range []string{"avg(", "sum(", "max(", "min("} {
			if strings.HasPrefix(strings.ToLower(name), n) {
				fieldType = "float"
			}
		}

		if as != name {
			cleanupName = strcase.ToCamel(as)
		}

		columns = append(columns, &core.SQLColumn{
			Name:     name,
			As:       as,
			Ref:      strcase.ToCamel(cleanupName),
			Type:     fieldType,
			TypeSql:  fieldSqlType,
			Nullable: false,
			Order:    i * 1000000,
		})
	}

	return columns
}

func parseTable(schema *core.SQLSchema, stmt *tree.CreateTable) {
	tableSchema := &core.SQLTable{
		Name:    stmt.Table.TableName.Normalize(),
		Columns: make(map[string]*core.SQLColumn),
		Order:   len(schema.Tables) + 1,
	}

	for _, def := range stmt.Defs {
		switch def := def.(type) {
		case *tree.ColumnTableDef:
			col := &core.SQLColumn{
				Name:     def.Name.Normalize(),
				Type:     def.Type.String(),
				TypeSql:  def.Type.SQLString(),
				Nullable: def.Nullable.Nullability != tree.NotNull,
				Order:    len(tableSchema.Columns) + 1,
			}

			tableSchema.Columns[def.Name.Normalize()] = col

			if col.TypeSql == "STRING" {
				col.TypeSql = "TEXT"
			}

			col.HasDefault = def.HasDefaultExpr()
			if def.IsSerial {
				col.HasDefault = true
			}

			if def.Unique {
				tableSchema.Constraints = append(tableSchema.Constraints, &core.SQLTableConstraint{
					Name:    def.UniqueConstraintName.Normalize(),
					Columns: []string{def.Name.Normalize()},
					Type:    "UNIQUE",
				})
				tableSchema.Indexes = append(tableSchema.Indexes, &core.SQLTableIndex{
					Name:    def.UniqueConstraintName.Normalize(),
					Columns: []string{def.Name.Normalize()},
				})
			}

			if def.PrimaryKey.IsPrimaryKey {
				tableSchema.Columns[def.Name.Normalize()].Nullable = false
				tableSchema.Constraints = append(tableSchema.Constraints, &core.SQLTableConstraint{
					Name:    "",
					Columns: []string{def.Name.Normalize()},
					Type:    "PRIMARY",
				})
				tableSchema.Indexes = append(tableSchema.Indexes, &core.SQLTableIndex{
					Name:    "",
					Columns: []string{def.Name.Normalize()},
				})
			}

			if def.References.Table != nil {
				tableSchema.References = append(tableSchema.References, &core.SQLTableReference{
					Name:         def.References.ConstraintName.Normalize(),
					Columns:      []string{def.Name.Normalize()},
					Table:        def.References.Table.TableName.Normalize(),
					TableColumns: []string{def.References.Col.Normalize()},
				})
			}
		case *tree.UniqueConstraintTableDef:
			tableSchema.Constraints = append(tableSchema.Constraints, &core.SQLTableConstraint{
				Name:    def.Name.Normalize(),
				Columns: NamesToStrings(def.Columns),
				Type:    "UNIQUE",
			})
			tableSchema.Indexes = append(tableSchema.Indexes, &core.SQLTableIndex{
				Name:    def.Name.Normalize(),
				Columns: NamesToStrings(def.Columns),
			})
		case *tree.CheckConstraintTableDef:
			tableSchema.Constraints = append(tableSchema.Constraints, &core.SQLTableConstraint{
				Name:    def.Name.Normalize(),
				Columns: []string{def.Expr.String()},
				Type:    "CHECK",
			})
		default:
			fmt.Printf("unknown def: %T\n", def)
		}
	}

	schema.Tables[tableSchema.Name] = tableSchema
}

func parseAlterTable(schema *core.SQLSchema, stmt *tree.AlterTable) {
	for _, cmd := range stmt.Cmds {
		switch cmd := cmd.(type) {
		case *tree.AlterTableDropColumn:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			delete(refTable.Columns, cmd.Column.Normalize())
		case *tree.AlterTableAlterColumnType:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			refTable.Columns[cmd.Column.Normalize()].Type = cmd.ToType.String()
		case *tree.AlterTableAddColumn:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			refTable.Columns[cmd.ColumnDef.Name.Normalize()] = &core.SQLColumn{
				Name:     cmd.ColumnDef.Name.Normalize(),
				Type:     cmd.ColumnDef.Type.String(),
				Nullable: cmd.ColumnDef.Nullable.Nullability != tree.NotNull,
			}
		case *tree.AlterTableRenameColumn:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]
			from := cmd.Column.Normalize()
			to := cmd.NewName.Normalize()
			refTable.Columns[to] = refTable.Columns[from]
			delete(refTable.Columns, from)

		case *tree.AlterTableDropConstraint:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			refTable.Constraints = slices.DeleteFunc(refTable.Constraints, func(c *core.SQLTableConstraint) bool {
				return c.Name == cmd.Constraint.Normalize()
			})

			refTable.Indexes = slices.DeleteFunc(refTable.Indexes, func(c *core.SQLTableIndex) bool {
				return c.Name == cmd.Constraint.Normalize()
			})

		case *tree.AlterTableAddConstraint:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			switch cmd.ConstraintDef.(type) {
			case *tree.UniqueConstraintTableDef:
				unique := cmd.ConstraintDef.(*tree.UniqueConstraintTableDef)
				refTable.Constraints = append(refTable.Constraints, &core.SQLTableConstraint{
					Name:    unique.Name.Normalize(),
					Columns: NamesToStrings(unique.Columns),
					Type:    "UNIQUE",
				})
				refTable.Indexes = append(refTable.Indexes, &core.SQLTableIndex{
					Name:    unique.Name.Normalize(),
					Columns: NamesToStrings(unique.Columns),
				})

			case *tree.ForeignKeyConstraintTableDef:
				fk := cmd.ConstraintDef.(*tree.ForeignKeyConstraintTableDef)

				refTable.References = append(refTable.References, &core.SQLTableReference{
					Name:         fk.Name.Normalize(),
					Columns:      NameListToStrings(fk.FromCols),
					Table:        fk.Table.TableName.Normalize(),
					TableColumns: NameListToStrings(fk.ToCols),
				})
			}
		}
	}
}

type FindTableDepsCtx struct {
	tables []string
	Tables []core.TableDeps
}
