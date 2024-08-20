package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/kefniark/mango-sql/internal"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Version:  "v0.0.1",
		Name:     "mangosql",
		HelpName: "MangoSQL",
		Usage:    "Generate a SQL Client from a SQL file or folder of SQL migrations",
		UsageText: `Syntax: mangosql [options] <source folder>
Example: mangosql --output db/file.go db/schema.sql`,
		Suggest:              true,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "database/client.go",
				Usage:   "Output file",
			},
			&cli.StringFlag{
				Name:    "package",
				Aliases: []string{"p"},
				Value:   "database",
				Usage:   "Go Package",
			},
			&cli.StringFlag{
				Name:    "driver",
				Aliases: []string{"d"},
				Value:   "pgx",
				Usage:   "SQL Driver",
			},
			&cli.StringFlag{
				Name:    "logger",
				Aliases: []string{"l"},
				Value:   "none",
				Usage:   "Logging library",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.NArg() <= 0 {
				return fmt.Errorf("missing source folder")
			}

			allowed_drivers := []string{"pq", "pgx", "sqlite"}
			driver := ctx.String("driver")
			if !slices.Contains(allowed_drivers, driver) {
				return fmt.Errorf("unknown driver, should be one of %v", allowed_drivers)
			}

			allowed_logger := []string{"none", "zap", "logrus", "zerolog", "console"}
			logger := ctx.String("logger")
			if !slices.Contains(allowed_logger, logger) {
				return fmt.Errorf("unknown logger, should be one of %v", allowed_logger)
			}

			name := ctx.Args().Get(0)
			return generate(GenerateOptions{
				Src:     name,
				Output:  ctx.String("output"),
				Package: ctx.String("package"),
				Driver:  driver,
				Logger:  logger,
			})
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type GenerateOptions struct {
	Src     string
	Output  string
	Package string
	Driver  string
	Logger  string
}

func generate(opts GenerateOptions) error {
	stat, err := os.Stat(opts.Src)
	if err != nil {
		return err
	}

	var sql string
	var queriesFilePath string
	var queriesSql string
	if stat.IsDir() {
		sql = parseMigrationFolder(opts.Src)

		queriesFilePath = path.Join(opts.Src, "queries.sql")
	} else {
		data, err := os.ReadFile(opts.Src)
		if err != nil {
			return err
		}
		sql = string(data)

		queriesFilePath = path.Join(path.Dir(opts.Src), "queries.sql")
	}

	schema, err := internal.ParseSchema(sql)
	if err != nil {
		return err
	}

	if _, err = os.Stat(queriesFilePath); err == nil {
		data, err := os.ReadFile(queriesFilePath)
		if err != nil {
			return err
		}
		queriesSql = string(data)
		err = internal.ParseQueries(schema, queriesSql)
		if err != nil {
			return err
		}
	}

	var b bytes.Buffer
	contents := bufio.NewWriter(&b)

	if err = internal.Generate(schema, contents, opts.Package, opts.Driver, opts.Logger); err != nil {
		return err
	}

	folder := path.Dir(opts.Output)
	file := path.Base(opts.Output)

	stat, err = os.Stat(opts.Output)
	if err == nil && stat.IsDir() {
		folder = opts.Output
		file = "client.go"
	}

	if err = os.MkdirAll(folder, os.ModeAppend); err != nil {
		return err
	}

	f, err := os.Create(path.Join(folder, file))
	if err != nil {
		return err
	}

	defer f.Close()

	contents.Flush()

	formatted, err := format.Source([]byte((b.String())))
	if err != nil {
		return err
	}

	_, err = f.WriteString(string(formatted))
	fmt.Printf("Generated %s\n", path.Join(folder, file))

	return err
}

func parseMigrationFolder(folderName string) string {
	entries, err := os.ReadDir(folderName)
	if err != nil {
		panic(err)
	}

	sql := []string{}
	for _, entries := range entries {
		if entries.IsDir() {
			continue
		}

		fileName := path.Join(folderName, entries.Name())
		data, err := os.ReadFile(fileName)
		if err != nil {
			panic(err)
		}

		entry := parseMigrationFile(string(data))
		sql = append(sql, strings.TrimSpace(entry))
	}

	return strings.Join(sql, "\n")
}

var (
	parseGooseMetaUp   = regexp.MustCompile(`-- \+goose Up`)
	parseGooseMetaDown = regexp.MustCompile(`-- \+goose Down`)
)

func parseMigrationFile(content string) string {
	up := parseGooseMetaUp.FindStringIndex(content)
	down := parseGooseMetaDown.FindStringIndex(content)

	if len(up) == 0 {
		return ""
	}

	if len(down) == 0 {
		return content[up[1]:]
	}

	if up[0] < down[0] {
		return content[up[1]:down[0]]
	}

	return content[down[1]:up[0]]
}
