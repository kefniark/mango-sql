package generator

import (
	"fmt"
	"regexp"

	"github.com/kefniark/mango-sql/internal/core"
)

var postgresType = map[string]string{
	"int":     "int64",
	"integer": "int64",
	// "int2":    "int16",
	// "int4":    "int32",
	// "int8":    "int64",

	"decimal": "float64",
	"float":   "float64",
	// "float2":  "float16",
	// "float4":  "float32",
	// "float8":  "float64",
	"bytes":       "[]byte",
	"string":      "string",
	"text":        "string",
	"char":        "string",
	"varchar":     "string",
	"date":        "time.Time",
	"time":        "time.Time",
	"timestamp":   "time.Time",
	"timestamptz": "time.Time",
	"uuid":        "uuid.UUID",
	"bool":        "bool",
	"json":        "interface{}",
	"jsonb":       "interface{}",
}

var parseType = regexp.MustCompile(`(?P<Type>[a-zA-Z]+)(?P<Accuracy>\d*)?(?P<Array>[\[\]]*)?`)

func getColumnType(column *core.SQLColumn) string {
	fieldType := column.Type
	fieldAccuracy := ""
	isArray := false

	for _, res := range parseType.FindAllStringSubmatch(fieldType, -1) {
		fieldType = res[1]
		fieldAccuracy = res[2]
		if res[3] != "" {
			isArray = true
		}
	}

	newType := "string"
	if val, ok := postgresType[fieldType+fieldAccuracy]; ok {
		newType = val
	} else if val2, ok2 := postgresType[fieldType]; ok2 {
		newType = val2
	} else {
		fmt.Println("Unknown type", column.Name, column.Table, column.Type)
	}

	if isArray && column.Nullable {
		return fmt.Sprintf("*[]%s", newType)
	} else if isArray {
		return fmt.Sprintf("[]%s", newType)
	} else if column.Nullable {
		return fmt.Sprintf("*%s", newType)
	}
	return newType
}
