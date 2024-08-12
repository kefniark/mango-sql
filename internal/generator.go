package internal

import (
	"fmt"
	"io"

	"github.com/kefniark/mango-sql/internal/core"
	"github.com/kefniark/mango-sql/internal/postgres"
)

func Generate(schema *core.SQLSchema, contents io.Writer, pkg string, driver string) error {
	switch driver {
	case "pq", "pgx":
		return postgres.Generate(schema, contents, pkg, driver)
	}

	return fmt.Errorf("driver %s not supported", driver)
}
