package postgres

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

func GetFilterMethods(tables []*PostgresTable) []FilterMethod {
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
			method.Filters = append(method.Filters,
				SelectFilter{
					Model:   t,
					Name:    "Equal",
					Comment: `Only include Records with a field specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s = $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "NotEqual",
					Comment: `Exclude Records with a field specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s != $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "In",
					Comment: `Only include Records with a field contains in a set of values`,
					Sql:     `.Where(fmt.Sprintf("%s.%s = ANY($1)", f.table, f.field), pq.Array(args))`,
					Args:    []string{"args ...T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "NotIn",
					Comment: `Exclude Records with a field not contains in a set of values`,
					Sql:     `.Where(fmt.Sprintf("%s.%s != ANY($1)", f.table, f.field), pq.Array(args))`,
					Args:    []string{"args ...T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "IsNull",
					Comment: `Only include Records with a field has undefined value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s IS NULL", f.table, f.field))`,
					Args:    []string{},
				},
				SelectFilter{
					Model:   t,
					Name:    "IsNotNull",
					Comment: `Only include Records with a field has defined values`,
					Sql:     `.Where(fmt.Sprintf("%s.%s IS NOT NULL", f.table, f.field))`,
					Args:    []string{},
				},
				SelectFilter{
					Model:   t,
					Name:    "OrderAsc",
					Comment: `Sort Records in ASC order`,
					Sql:     `.OrderBy(fmt.Sprintf("%s.%s ASC", f.table, f.field))`,
					Args:    []string{},
				},
				SelectFilter{
					Model:   t,
					Name:    "OrderDesc",
					Comment: `Sort Records in DESC order`,
					Sql:     `.OrderBy(fmt.Sprintf("%s.%s DESC", f.table, f.field))`,
					Args:    []string{},
				},
			)
		}

		if t == "FilterStringField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model:   t,
					Name:    "Like",
					Comment: `Only include Records with a field contains a specific value (use % as wildcard)`,
					Sql:     `.Where(fmt.Sprintf("%s.%s LIKE $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "NotLike",
					Comment: `Exclude Records with a field contains a specific value  (use % as wildcard)`,
					Sql:     `.Where(fmt.Sprintf("%s.%s NOT LIKE $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
			)
		}

		if t == "FilterNumericField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model:   t,
					Name:    "GreaterThan",
					Comment: `Only include Records with a field greater than a specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s > $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "GreaterThanOrEqual",
					Comment: `Only include Records with a field greater or equal than a specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s >= $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "LesserThan",
					Comment: `Only include Records with a field lesser than a specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s < $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "LesserThanOrEqual",
					Comment: `Only include Records with a field lesser or equal than a specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s <= $1", f.table, f.field), arg)`,
					Args:    []string{"arg T"},
				},
				SelectFilter{
					Model:   t,
					Name:    "Between",
					Comment: `Only include Records with a field value in a specified range`,
					Sql:     `.Where(fmt.Sprintf("%s.%s BETWEEN $1 AND $2", f.table, f.field), from, to)`,
					Args:    []string{"from T", "to T"},
				},
			)
		}

		if t == "FilterArrayField" {
			method.Filters = append(method.Filters,
				SelectFilter{
					Model:   t,
					Name:    "Contains",
					Comment: `Check if a array field contains a specific value`,
					Sql:     `.Where(fmt.Sprintf("%s.%s @> $1", f.table, f.field), pq.Array(args))`,
					Args:    []string{"args T"},
				},
			)
		}

		methods = append(methods, method)
	}
	return methods
}
