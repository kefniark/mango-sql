package generator

import (
	"slices"
	"strings"
)

const (
	FilterNumericField = "FilterNumericField"
	FilterStringField  = "FilterStringField"
	FilterArrayField   = "FilterArrayField"
	FilterGenericField = "FilterGenericField"
)

func GetNormalizedTypeFilter(col *PostgresColumn) string {
	if strings.HasPrefix(col.Type, "[]") {
		return FilterArrayField
	} else if strings.Contains(strings.ToLower(col.Type), "string") {
		return FilterStringField
	} else if strings.Contains(strings.ToLower(col.Type), "int") {
		return FilterNumericField
	} else if strings.Contains(strings.ToLower(col.Type), "float") {
		return FilterNumericField
	} else if strings.Contains(strings.ToLower(col.Type), "time") {
		return FilterNumericField
	}
	return FilterGenericField
}

func (table *PostgresTable) GetFieldFilters() []SelectFieldFilter {
	filters := []SelectFieldFilter{}
	for _, col := range table.Columns {
		filters = append(filters, SelectFieldFilter{
			Model:     table.NameNormalized,
			Table:     table.Name,
			Name:      col.NameNormalized,
			Field:     col.Name,
			FieldType: col.Type,
			Type:      GetNormalizedTypeFilter(col),
		})
	}
	return filters
}

func GetFilterMethods(tables []*PostgresTable, driver string) []FilterMethod {
	supportedFilters := []string{"FilterGenericField"}
	for _, table := range tables {
		for _, col := range table.Columns {
			t := GetNormalizedTypeFilter(col)
			if slices.Contains(supportedFilters, t) {
				continue
			}
			supportedFilters = append(supportedFilters, t)
		}
	}

	methods := []FilterMethod{}
	for _, t := range supportedFilters {
		method := FilterMethod{Name: t}

		if t == "FilterGenericField" {
			if driver == "sqlite" {
				method.Filters = append(method.Filters,
					SelectFilter{
						Model: t,
						Name:  "In",
						Pre: `jsonArgs, err := json.Marshal(args)
							if err != nil {
								return f.IsNull()
							}
							sql := fmt.Sprintf("%s.%s IN (SELECT value FROM json_each(?))", f.table, f.field)
							`,
						Comment: `Only include Records with a field contains in a set of values`,
						Sql:     `.Where(sql, jsonArgs)`,
						Args:    []string{"args ...T"},
					},
					SelectFilter{
						Model: t,
						Name:  "NotIn",
						Pre: `jsonArgs, err := json.Marshal(args)
							if err != nil {
								return f.IsNull()
							}
							sql := fmt.Sprintf("%s.%s NOT IN (SELECT value FROM json_each(?))", f.table, f.field)
							`,
						Comment: `Exclude Records with a field not contains in a set of values`,
						Sql:     `.Where(sql, jsonArgs)`,
						Args:    []string{"args ...T"},
					},
				)
			} else if driver == "mysql" || driver == "mariadb" {
				method.Filters = append(method.Filters,
					SelectFilter{
						Model: t,
						Name:  "In",
						Pre: `sql := fmt.Sprintf("%s.%s IN(?)", f.table, f.field)
						`,
						Comment: `Only include Records with a field contains in a set of values`,
						PreSql: `query, args, _ := sqlx.In(sql, args)
						`,
						Sql:  `.Where(sqlx.Rebind(sqlx.QUESTION, query), args...)`,
						Args: []string{"args ...T"},
					},
					SelectFilter{
						Model: t,
						Name:  "NotIn",
						Pre: `sql := fmt.Sprintf("%s.%s NOT IN(?)", f.table, f.field)
						`,
						Comment: `Exclude Records with a field not contains in a set of values`,
						PreSql: `query, args, _ := sqlx.In(sql, args)
						`,
						Sql:  `.Where(sqlx.Rebind(sqlx.QUESTION, query), args...)`,
						Args: []string{"args ...T"},
					},
				)
			} else {
				method.Filters = append(method.Filters,
					SelectFilter{
						Model: t,
						Name:  "In",
						Pre: `sql := fmt.Sprintf("%s.%s = ANY(?)", f.table, f.field)
						`,
						Comment: `Only include Records with a field contains in a set of values`,
						Sql:     `.Where(sql, pq.Array(args))`,
						Args:    []string{"args ...T"},
					},
					SelectFilter{
						Model: t,
						Name:  "NotIn",
						Pre: `sql := fmt.Sprintf("%s.%s != ANY(?)", f.table, f.field)
						`,
						Comment: `Exclude Records with a field not contains in a set of values`,
						Sql:     `.Where(sql, pq.Array(args))`,
						Args:    []string{"args ...T"},
					},
				)
			}

			method.Filters = append(method.Filters,
				SelectFilter{
					Model: t,
					Name:  "Equal",
					Pre: `sql := fmt.Sprintf("%s.%s = ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "NotEqual",
					Pre: `sql := fmt.Sprintf("%s.%s != ?", f.table, f.field)
					`,
					Comment: `Exclude Records with a field specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},

				SelectFilter{
					Model: t,
					Name:  "IsNull",
					Pre: `sql := fmt.Sprintf("%s.%s IS NULL", f.table, f.field)
					`,
					Comment: `Only include Records with a field has undefined value`,
					Sql:     `.Where(sql)`,
					Args:    []string{},
				},
				SelectFilter{
					Model: t,
					Name:  "IsNotNull",
					Pre: `sql := fmt.Sprintf("%s.%s IS NOT NULL", f.table, f.field)
					`,
					Comment: `Only include Records with a field has defined values`,
					Sql:     `.Where(sql)`,
					Args:    []string{},
				},
				SelectFilter{
					Model: t,
					Name:  "OrderAsc",
					Pre: `sql := fmt.Sprintf("%s.%s ASC", f.table, f.field)
					`,
					Comment: `Sort Records in ASC order`,
					Sql:     `.OrderBy(sql)`,
					Args:    []string{},
				},
				SelectFilter{
					Model: t,
					Name:  "OrderDesc",
					Pre: `sql := fmt.Sprintf("%s.%s DESC", f.table, f.field)
					`,
					Comment: `Sort Records in DESC order`,
					Sql:     `.OrderBy(sql)`,
					Args:    []string{},
				},
			)
		}

		if t == "FilterStringField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model: t,
					Name:  "Like",
					Pre: `sql := fmt.Sprintf("%s.%s LIKE ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field contains a specific value (use % as wildcard)`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "NotLike",
					Pre: `sql := fmt.Sprintf("%s.%s NOT LIKE ?", f.table, f.field)
					`,
					Comment: `Exclude Records with a field contains a specific value  (use % as wildcard)`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
			)
		}

		if t == "FilterNumericField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model: t,
					Name:  "GreaterThan",
					Pre: `sql := fmt.Sprintf("%s.%s > ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field greater than a specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "GreaterThanOrEqual",
					Pre: `sql := fmt.Sprintf("%s.%s >= ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field greater or equal than a specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "LesserThan",
					Pre: `sql := fmt.Sprintf("%s.%s < ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field lesser than a specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "LesserThanOrEqual",
					Pre: `sql := fmt.Sprintf("%s.%s <= ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field lesser or equal than a specific value`,
					Sql:     `.Where(sql, arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model: t,
					Name:  "Between",
					Pre: `sql := fmt.Sprintf("%s.%s BETWEEN ? AND ?", f.table, f.field)
					`,
					Comment: `Only include Records with a field value in a specified range`,
					Sql:     `.Where(sql, from, to)`,
					Args:    []string{"from T", "to T"},
				},
			)
		}

		if t == "FilterArrayField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model: t,
					Name:  "Contains",
					Pre: `sql := fmt.Sprintf("%s.%s @> ?", f.table, f.field)
					`,
					Comment: `Check if a array field contains a specific value`,
					Sql:     `.Where(sql, pq.Array(args))`,
					Args:    []string{"args T"},
				},
			)
		}

		methods = append(methods, method)
	}
	return methods
}
