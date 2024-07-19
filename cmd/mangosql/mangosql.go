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
	"strings"

	"github.com/kefniark/mango-sql/internal"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:     "mangosql",
		HelpName: "MangoSQL",
		Usage:    "Generate a SQL Client from a SQL file or folder of SQL migrations",
		UsageText: `Syntax: mangosql [options] <source folder>
Example: mangosql --output db/file.go db/schema.sql`,
		Suggest: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "output",
				Value: "database/client.go",
				Usage: "Output file",
			},
			&cli.StringFlag{
				Name:  "package",
				Value: "database",
				Usage: "Go Package",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.NArg() <= 0 {
				return fmt.Errorf("missing source folder")
			}

			name := ctx.Args().Get(0)
			return generate(GenerateOptions{
				Src:     name,
				Output:  ctx.String("output"),
				Package: ctx.String("package"),
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
}

func generate(opts GenerateOptions) error {
	stat, err := os.Stat(opts.Output)
	if err != nil {
		return err
	}

	var sql string
	if stat.IsDir() {
		sql = parseMigrationFolder(opts.Src)
	} else {
		data, err := os.ReadFile(opts.Src)
		if err != nil {
			return err
		}
		sql = string(data)
	}

	schema, err := internal.ParseSchema(sql)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	contents := bufio.NewWriter(&b)

	if err = internal.Generate(schema, contents, opts.Package); err != nil {
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

var parseGooseMetaUp = regexp.MustCompile(`-- \+goose Up`)
var parseGooseMetaDown = regexp.MustCompile(`-- \+goose Down`)

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
