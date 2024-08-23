package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/kefniark/mango-sql/internal/core"
)

func Parse(url string) (*core.SQLSchema, error) {
	database, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return nil, err
	}

	db := New(database)

	tableNames, err := db.Queries.TableNames()
	if err != nil {
		return nil, err
	}

	schema := &core.SQLSchema{
		Tables: make(map[string]*core.SQLTable),
	}

	for i, tableName := range tableNames {
		table := &core.SQLTable{
			Name:    *tableName.TableName,
			Columns: map[string]*core.SQLColumn{},
			Order:   i,
		}

		cols, err := db.Queries.TableColumns(
			func(query SelectBuilder) SelectBuilder {
				return query.Where("table_name = ?", tableName.TableName)
			},
		)
		if err != nil {
			return nil, err
		}

		for _, col := range cols {
			table.Columns[*col.ColumnName] = &core.SQLColumn{
				Name:       *col.ColumnName,
				HasDefault: col.ColumnDefault != nil,
				Type:       "string",
				TypeSql:    *col.UdtName,
				Nullable:   *col.IsNullable == "true",
			}
		}

		schema.Tables[*tableName.TableName] = table
	}

	return schema, nil
}
