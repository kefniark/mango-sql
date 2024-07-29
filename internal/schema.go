package internal

import "fmt"

type SQLSchema struct {
	Tables  map[string]*SQLTable
	Queries []SQLQuery
}

type SQLTable struct {
	Name    string
	Columns map[string]*SQLColumn

	Constraints []*SQLTableConstraint
	Indexes     []*SQLTableIndex
	References  []*SQLTableReference
	Referenced  []*SQLTableReference

	Order int
}

type SQLQuery struct {
	Query  string
	Method string
	Name   string

	Select         string
	SelectOriginal string
	SelectFields   []*SQLColumn

	From    string
	Where   string
	GroupBy []string
	Having  string
}

type SQLColumn struct {
	Name       string
	As         string
	Ref        string
	Type       string
	TypeSql    string
	Nullable   bool
	Table      string
	TableAs    string
	HasDefault bool

	Order int
}

type SQLTableConstraint struct {
	Name    string
	Columns []string
	Type    string
}

type SQLTableIndex struct {
	Name    string
	Columns []string
}

type SQLTableReference struct {
	Name         string
	Columns      []string
	Table        string
	TableColumns []string
}

func (s *SQLSchema) Debug() {
	for name, table := range s.Tables {
		fmt.Printf("table: %s\n", name)
		for _, column := range table.Columns {
			fmt.Printf("  column: %s %s\n", column.Name, column.Type)
		}

		for _, constraint := range table.Constraints {
			fmt.Printf("  constraint: %s %s %s\n", constraint.Name, constraint.Columns, constraint.Type)
		}

		for _, index := range table.Indexes {
			fmt.Printf("  index: %s %s\n", index.Name, index.Columns)
		}

		for _, ref := range table.References {
			fmt.Printf("  ref: %s %s => %s(%s)\n", ref.Name, ref.Columns, ref.Table, ref.TableColumns)
		}
	}
}
