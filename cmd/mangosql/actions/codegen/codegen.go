package codegen

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/kefniark/mango-sql/cmd/mangosql/input"
	"github.com/kefniark/mango-sql/internal"
	"github.com/kefniark/mango-sql/internal/generator"
	"github.com/urfave/cli/v2"
)

func Action(ctx *cli.Context) error {
	if ctx.NArg() <= 0 {
		return errors.New("missing source folder")
	}

	allowedDrivers := []string{"pq", "pgx", "sqlite", "mysql", "mariadb"}
	driver := ctx.String("driver")
	if !slices.Contains(allowedDrivers, driver) {
		return fmt.Errorf("unknown driver, should be one of %v", allowedDrivers)
	}

	allowedLoggers := []string{"none", "zap", "logrus", "zerolog", "console"}
	logger := ctx.String("logger")
	if !slices.Contains(allowedLoggers, logger) {
		return fmt.Errorf("unknown logger, should be one of %v", allowedLoggers)
	}

	name := ctx.Args().Get(0)
	return generate(generateOptions{
		Src:     name,
		Output:  ctx.String("output"),
		Inline:  ctx.Bool("inline"),
		Package: ctx.String("package"),
		Driver:  driver,
		Logger:  logger,
	})
}

type generateOptions struct {
	Src     string
	Output  string
	Inline  bool
	Package string
	Driver  string
	Logger  string
}

func generate(opts generateOptions) error {
	// find schema
	sql, err := input.ParseInputSchema(opts.Src)
	if err != nil {
		return err
	}

	// parse schema
	schema, err := internal.ParseSchema(sql)
	if err != nil {
		fmt.Printf("schema parsing error: %+v\n", err)
		return err
	}

	// find and parse queries
	if queries, err := input.ParseInputQueries(opts.Src); err == nil {
		err = internal.ParseQueries(schema, queries)
		if err != nil {
			return err
		}
	}

	var b bytes.Buffer
	contents := bufio.NewWriter(&b)

	if err = generator.Generate(schema, contents, opts.Package, opts.Driver, opts.Logger); err != nil {
		return err
	}

	folder := path.Dir(opts.Output)
	file := path.Base(opts.Output)

	stat, err := os.Stat(opts.Output)
	if err == nil && stat.IsDir() {
		folder = opts.Output
		file = "client.go"
	}

	if err = os.MkdirAll(folder, os.ModeAppend); err != nil {
		return err
	}

	contents.Flush()
	formatted, err := format.Source([]byte((b.String())))
	if err != nil {
		return err
	}

	if opts.Inline {
		return nil
	}

	f, err := os.Create(path.Join(folder, file))
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(formatted)
	if path, err := filepath.Abs(path.Join(folder, file)); err == nil {
		fmt.Printf("Generated %s\n", path)
	}

	return err
}
