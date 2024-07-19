package internal

import (
	"fmt"
	"slices"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
)

func ParseSchema(sql string) (*SQLSchema, error) {
	stmts, err := parser.Parse(sql)
	if err != nil {
		return nil, err
	}

	schema := &SQLSchema{
		Tables: make(map[string]*SQLTable),
	}

	w := &walk.AstWalker{
		Fn: func(ctx interface{}, node interface{}) (stop bool) {
			// log.Printf("node type %T", node)
			switch stmt := node.(type) {
			case *tree.CreateTable:
				parseTable(schema, stmt)
			case *tree.CreateIndex:
				// fmt.Println("index name: ", stmt.Name)
				refTable := schema.Tables[stmt.Table.TableName.Normalize()]
				if stmt.Unique {
					refTable.Constraints = append(refTable.Constraints, &SQLTableConstraint{
						Name:    stmt.Name.Normalize(),
						Columns: NamesToStrings(stmt.Columns),
						Type:    "UNIQUE",
					})
				}
				refTable.Indexes = append(refTable.Indexes, &SQLTableIndex{
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

	_, _ = w.Walk(stmts, nil)

	return schema, nil
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

func parseTable(schema *SQLSchema, stmt *tree.CreateTable) {
	tableSchema := &SQLTable{
		Name:    stmt.Table.TableName.Normalize(),
		Columns: make(map[string]*SQLColumn),
		Order:   len(schema.Tables) + 1,
	}

	for _, def := range stmt.Defs {
		switch def := def.(type) {
		case *tree.ColumnTableDef:
			col := &SQLColumn{
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

			if def.Unique {
				tableSchema.Constraints = append(tableSchema.Constraints, &SQLTableConstraint{
					Name:    def.UniqueConstraintName.Normalize(),
					Columns: []string{def.Name.Normalize()},
					Type:    "UNIQUE",
				})
				tableSchema.Indexes = append(tableSchema.Indexes, &SQLTableIndex{
					Name:    def.UniqueConstraintName.Normalize(),
					Columns: []string{def.Name.Normalize()},
				})
			}

			if def.PrimaryKey.IsPrimaryKey {
				tableSchema.Columns[def.Name.Normalize()].Nullable = false
				tableSchema.Constraints = append(tableSchema.Constraints, &SQLTableConstraint{
					Name:    "",
					Columns: []string{def.Name.Normalize()},
					Type:    "PRIMARY",
				})
				tableSchema.Indexes = append(tableSchema.Indexes, &SQLTableIndex{
					Name:    "",
					Columns: []string{def.Name.Normalize()},
				})
			}

			if def.References.Table != nil {
				tableSchema.References = append(tableSchema.References, &SQLTableReference{
					Name:         def.References.ConstraintName.Normalize(),
					Columns:      []string{def.Name.Normalize()},
					Table:        def.References.Table.TableName.Normalize(),
					TableColumns: []string{def.References.Col.Normalize()},
				})
			}
		case *tree.UniqueConstraintTableDef:
			tableSchema.Constraints = append(tableSchema.Constraints, &SQLTableConstraint{
				Name:    def.Name.Normalize(),
				Columns: NamesToStrings(def.Columns),
				Type:    "UNIQUE",
			})
			tableSchema.Indexes = append(tableSchema.Indexes, &SQLTableIndex{
				Name:    def.Name.Normalize(),
				Columns: NamesToStrings(def.Columns),
			})
		case *tree.CheckConstraintTableDef:
			tableSchema.Constraints = append(tableSchema.Constraints, &SQLTableConstraint{
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

func parseAlterTable(schema *SQLSchema, stmt *tree.AlterTable) {
	// fmt.Println("alter table")
	for _, cmd := range stmt.Cmds {
		// fmt.Println("cmd type: ", cmd)
		switch cmd := cmd.(type) {
		case *tree.AlterTableDropColumn:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			delete(refTable.Columns, cmd.Column.Normalize())
		case *tree.AlterTableAlterColumnType:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			refTable.Columns[cmd.Column.Normalize()].Type = cmd.ToType.String()
		case *tree.AlterTableAddColumn:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			refTable.Columns[cmd.ColumnDef.Name.Normalize()] = &SQLColumn{
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

			refTable.Constraints = slices.DeleteFunc(refTable.Constraints, func(c *SQLTableConstraint) bool {
				return c.Name == cmd.Constraint.Normalize()
			})

			refTable.Indexes = slices.DeleteFunc(refTable.Indexes, func(c *SQLTableIndex) bool {
				return c.Name == cmd.Constraint.Normalize()
			})

		case *tree.AlterTableAddConstraint:
			refTable := schema.Tables[stmt.Table.ToTableName().TableName.Normalize()]

			switch cmd.ConstraintDef.(type) {
			case *tree.UniqueConstraintTableDef:
				unique := cmd.ConstraintDef.(*tree.UniqueConstraintTableDef)
				refTable.Constraints = append(refTable.Constraints, &SQLTableConstraint{
					Name:    unique.Name.Normalize(),
					Columns: NamesToStrings(unique.Columns),
					Type:    "UNIQUE",
				})
				refTable.Indexes = append(refTable.Indexes, &SQLTableIndex{
					Name:    unique.Name.Normalize(),
					Columns: NamesToStrings(unique.Columns),
				})

			case *tree.ForeignKeyConstraintTableDef:
				fk := cmd.ConstraintDef.(*tree.ForeignKeyConstraintTableDef)

				refTable.References = append(refTable.References, &SQLTableReference{
					Name:         fk.Name.Normalize(),
					Columns:      NameListToStrings(fk.FromCols),
					Table:        fk.Table.TableName.Normalize(),
					TableColumns: NameListToStrings(fk.ToCols),
				})
			}

		}
	}
}
