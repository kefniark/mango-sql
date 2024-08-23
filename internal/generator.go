package internal

import (
	"fmt"
	"io"

	"github.com/kefniark/mango-sql/internal/core"
	"github.com/kefniark/mango-sql/internal/generator"
)

func Generate(schema *core.SQLSchema, contents io.Writer, pkg string, driver string, logger string) error {
	switch driver {
	case "sqlite", "pq", "pgx":
		return generator.Generate(schema, contents, pkg, driver, logger)
	}

	return fmt.Errorf("driver %s not supported", driver)
}
