package generator

import (
	"slices"
	"strings"

	"github.com/kefniark/mango-sql/internal/core"
)

const (
	FilterNumericField = "FilterNumericField"
	FilterStringField  = "FilterStringField"
	FilterArrayField   = "FilterArrayField"
	FilterGenericField = "FilterGenericField"
)

func GetNormalizedTypeFilter(col *PostgresColumn) string {
	switch {
	case strings.HasPrefix(col.Type, "[]"):
		return FilterArrayField
	case strings.Contains(strings.ToLower(col.Type), "string"):
		return FilterStringField
	case strings.Contains(strings.ToLower(col.Type), "int"), strings.Contains(strings.ToLower(col.Type), "float"), strings.Contains(strings.ToLower(col.Type), "time"):
		return FilterNumericField
	default:
		return FilterGenericField
	}
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

		addFiltersIn(driver, t, &method)
		addFiltersCompare(t, &method)
		addFiltersLike(t, &method)
		addFiltersMathCompare(t, &method)
		addFiltersArray(t, &method)

		methods = append(methods, method)
	}
	return methods
}

func addFiltersCompare(t string, method *FilterMethod) {
	if t != "FilterGenericField" {
		return
	}

	method.Filters = append(method.Filters,
		SelectFilter{
			Model: t,
			Name:  "Equal",
			Pre: `sql := fmt.Sprintf("%s.%s = ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "NotEqual",
			Pre: `sql := fmt.Sprintf("%s.%s != ?", f.table, f.field)
			`,
			Comment: `Exclude Records with a field specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "IsNull",
			Pre: `sql := fmt.Sprintf("%s.%s IS NULL", f.table, f.field)
			`,
			Comment: `Only include Records with a field has undefined value`,
			SQL:     `.Where(sql)`,
			Args:    []string{},
		},
		SelectFilter{
			Model: t,
			Name:  "IsNotNull",
			Pre: `sql := fmt.Sprintf("%s.%s IS NOT NULL", f.table, f.field)
			`,
			Comment: `Only include Records with a field has defined values`,
			SQL:     `.Where(sql)`,
			Args:    []string{},
		},
		SelectFilter{
			Model: t,
			Name:  "OrderAsc",
			Pre: `sql := fmt.Sprintf("%s.%s ASC", f.table, f.field)
			`,
			Comment: `Sort Records in ASC order`,
			SQL:     `.OrderBy(sql)`,
			Args:    []string{},
		},
		SelectFilter{
			Model: t,
			Name:  "OrderDesc",
			Pre: `sql := fmt.Sprintf("%s.%s DESC", f.table, f.field)
			`,
			Comment: `Sort Records in DESC order`,
			SQL:     `.OrderBy(sql)`,
			Args:    []string{},
		},
	)
}

func addFiltersArray(t string, method *FilterMethod) {
	if t != "FilterArrayField" {
		return
	}

	method.Filters = append(method.Filters,
		SelectFilter{
			Model: t,
			Name:  "Contains",
			Pre: `sql := fmt.Sprintf("%s.%s @> ?", f.table, f.field)
			`,
			Comment: `Check if a array field contains a specific value`,
			SQL:     `.Where(sql, pq.Array(args))`,
			Args:    []string{"args T"},
		},
	)
}

func addFiltersMathCompare(t string, method *FilterMethod) {
	if t != "FilterNumericField" {
		return
	}

	method.Filters = append(method.Filters,
		SelectFilter{
			Model: t,
			Name:  "GreaterThan",
			Pre: `sql := fmt.Sprintf("%s.%s > ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field greater than a specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "GreaterThanOrEqual",
			Pre: `sql := fmt.Sprintf("%s.%s >= ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field greater or equal than a specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "LesserThan",
			Pre: `sql := fmt.Sprintf("%s.%s < ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field lesser than a specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "LesserThanOrEqual",
			Pre: `sql := fmt.Sprintf("%s.%s <= ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field lesser or equal than a specific value`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "Between",
			Pre: `sql := fmt.Sprintf("%s.%s BETWEEN ? AND ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field value in a specified range`,
			SQL:     `.Where(sql, from, to)`,
			Args:    []string{"from T", "to T"},
		},
	)
}

func addFiltersLike(t string, method *FilterMethod) {
	if t != "FilterStringField" {
		return
	}

	method.Filters = append(method.Filters,
		SelectFilter{
			Model: t,
			Name:  "Like",
			Pre: `sql := fmt.Sprintf("%s.%s LIKE ?", f.table, f.field)
			`,
			Comment: `Only include Records with a field contains a specific value (use % as wildcard)`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
		SelectFilter{
			Model: t,
			Name:  "NotLike",
			Pre: `sql := fmt.Sprintf("%s.%s NOT LIKE ?", f.table, f.field)
			`,
			Comment: `Exclude Records with a field contains a specific value  (use % as wildcard)`,
			SQL:     `.Where(sql, arg)`,
			Args:    []string{"arg T"},
		},
	)
}

func addFiltersIn(driver string, t string, method *FilterMethod) {
	if t != "FilterGenericField" {
		return
	}

	if driver == core.DriverSqlite {
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
				SQL:     `.Where(sql, jsonArgs)`,
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
				SQL:     `.Where(sql, jsonArgs)`,
				Args:    []string{"args ...T"},
			},
		)
	} else if driver == core.DriverMysql || driver == core.DriverMariaDB {
		method.Filters = append(method.Filters,
			SelectFilter{
				Model: t,
				Name:  "In",
				Pre: `sql := fmt.Sprintf("%s.%s IN(?)", f.table, f.field)
				`,
				Comment: `Only include Records with a field contains in a set of values`,
				PreSQL: `query, args, _ := sqlx.In(sql, args)
				`,
				SQL:  `.Where(sqlx.Rebind(sqlx.QUESTION, query), args...)`,
				Args: []string{"args ...T"},
			},
			SelectFilter{
				Model: t,
				Name:  "NotIn",
				Pre: `sql := fmt.Sprintf("%s.%s NOT IN(?)", f.table, f.field)
				`,
				Comment: `Exclude Records with a field not contains in a set of values`,
				PreSQL: `query, args, _ := sqlx.In(sql, args)
				`,
				SQL:  `.Where(sqlx.Rebind(sqlx.QUESTION, query), args...)`,
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
				SQL:     `.Where(sql, pq.Array(args))`,
				Args:    []string{"args ...T"},
			},
			SelectFilter{
				Model: t,
				Name:  "NotIn",
				Pre: `sql := fmt.Sprintf("%s.%s != ANY(?)", f.table, f.field)
				`,
				Comment: `Exclude Records with a field not contains in a set of values`,
				SQL:     `.Where(sql, pq.Array(args))`,
				Args:    []string{"args ...T"},
			},
		)
	}
}
