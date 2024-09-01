package diagram

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"cdr.dev/slog"
	"github.com/kefniark/mango-sql/cmd/mangosql/input"
	"github.com/kefniark/mango-sql/internal"
	"github.com/kefniark/mango-sql/internal/core"
	"github.com/urfave/cli/v2"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
	"oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "diagram",
		Usage: "Generate a ERD diagram from DB schema",
		UsageText: `Syntax: mangosql diagram [options] <source folder>
Example: mangosql diagram db/schema.sql`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "diagram.svg",
				Usage:   "Output file",
			},
			&cli.StringFlag{
				Name:    "title",
				Aliases: []string{"t"},
				Value:   "Database",
				Usage:   "Name of the diagram",
			},
			&cli.StringFlag{
				Name:    "meta",
				Aliases: []string{"m"},
				Value:   "",
				Usage:   "Metadata about the diagram, separated by '|' (can be used to provide context, version, git commit, feature name, ...)",
			},
			&cli.BoolFlag{
				Name:    "sketch",
				Aliases: []string{"s"},
				Value:   false,
				Usage:   "Enable Sketch Mode",
			},
			&cli.BoolFlag{
				Name:    "dark",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "Enable Dark Mode",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.NArg() <= 0 {
				return errors.New("missing source folder")
			}

			return diagram(diagramOptions{
				Src:    ctx.Args().Get(0),
				Dark:   ctx.Bool("dark"),
				Sketch: ctx.Bool("sketch"),
				Output: ctx.String("output"),
				Title:  ctx.String("title"),
				Meta:   ctx.String("meta"),
			})
		},
	}
}

type diagramOptions struct {
	Src    string
	Output string
	Sketch bool
	Dark   bool
	Title  string
	Meta   string
}

func diagram(opts diagramOptions) error {
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

	ctx := log.With(context.Background(), slog.Make())

	ruler, _ := textmeasure.NewRuler()
	layoutResolver := func(_ string) (d2graph.LayoutGraph, error) {
		return d2dagrelayout.DefaultLayout, nil
	}
	center := true

	themeID := d2themescatalog.GrapeSoda.ID
	if opts.Dark {
		themeID = d2themescatalog.DarkMauve.ID
	}

	renderOpts := &d2svg.RenderOpts{
		ThemeID: &themeID,
		Sketch:  &opts.Sketch,
		Center:  &center,
	}
	compileOpts := &d2lib.CompileOptions{
		LayoutResolver: layoutResolver,
		Ruler:          ruler,
	}

	lines, err := generateSchemaD2(opts, schema)
	if err != nil {
		return err
	}

	diagram, _, _ := d2lib.Compile(ctx, lines, compileOpts, renderOpts)
	out, _ := d2svg.Render(diagram, renderOpts)

	folder := path.Dir(opts.Output)
	file := path.Base(opts.Output)

	stat, err := os.Stat(opts.Output)
	if err == nil && stat.IsDir() {
		folder = opts.Output
		file = "diagram.svg"
	}

	if err = os.MkdirAll(folder, os.ModeAppend); err != nil {
		return err
	}

	f, err := os.Create(path.Join(folder, file))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(out)
	if path, err := filepath.Abs(path.Join(folder, file)); err == nil {
		fmt.Printf("Generated %s\n", path)
	}
	return err
}

func generateSchemaD2(opts diagramOptions, schema *core.SQLSchema) (string, error) {
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	if opts.Title != "" {
		meta := []string{}
		for _, s := range strings.Split(opts.Meta, "|") {
			v := strings.TrimSpace(s)
			if v == "" {
				continue
			}
			meta = append(meta, fmt.Sprintf("* %s\n", v))
		}

		_, err := writer.WriteString(fmt.Sprintf(`title: |md
# %s
* Date: %s
%s| {near: bottom-left}
`, opts.Title, time.Now().Format("2006-01-02"), strings.Join(meta, "")))
		if err != nil {
			return "", err
		}
	}

	tables := maps.Values(schema.Tables)
	tableSorted := slices.SortedStableFunc(tables, func(i, j *core.SQLTable) int {
		return i.Order - j.Order
	})

	for _, t := range tableSorted {
		if _, err := writer.WriteString(fmt.Sprintf("%s: {\n", tableName(t.Name))); err != nil {
			return "", err
		}

		if _, err := writer.WriteString("	shape: sql_table\n"); err != nil {
			return "", err
		}

		renderColumn(t, t.Columns, writer)

		if _, err := writer.WriteString("}\n"); err != nil {
			return "", err
		}
	}

	renderRelations(schema.Tables, writer)

	writer.Flush()
	return b.String(), nil
}

func renderRelations(tables map[string]*core.SQLTable, writer *bufio.Writer) {
	refs := map[string]string{}
	for _, t := range tables {
		for _, r := range t.References {
			ref := fmt.Sprintf(`%s.%s -> %s.%s: {
	style.opacity: 1
	style.animated: true
}
`, tableName(t.Name), r.Columns[0], tableName(r.Table), r.TableColumns[0])
			refs[ref] = ""
		}
	}

	for ref := range refs {
		if _, err := writer.WriteString(ref); err != nil {
			continue
		}
	}
}

func renderColumn(t *core.SQLTable, columns map[string]*core.SQLColumn, writer *bufio.Writer) {
	cols := maps.Values(columns)
	colsSorted := slices.SortedStableFunc(cols, func(i, j *core.SQLColumn) int {
		return i.Order - j.Order
	})

	for _, c := range colsSorted {
		refs := []string{}
		var meta string

		for _, ro := range t.References {
			if c.Name != ro.Columns[0] {
				continue
			}
			meta = ` {constraint: foreign_key}`
			refs = append(refs, ro.Columns...)
		}

		for _, ri := range t.Referenced {
			if c.Name != ri.Columns[0] {
				continue
			}
			refs = append(refs, ri.Columns...)
		}
		for _, i := range t.Constraints {
			if c.Name != i.Columns[0] {
				continue
			}
			if i.Type == "PRIMARY" {
				meta = ` {constraint: primary_key}`
			} else if meta == "" && i.Type == "UNIQUE" {
				meta = ` {constraint: unique}`
			}
			refs = append(refs, i.Columns...)
		}

		if slices.Contains(refs, c.Name) {
			_, err := writer.WriteString(fmt.Sprintf("	%s: %s%s\n", c.Name, typeName(c.Type), meta))
			if err != nil {
				continue
			}
		} else {
			_, err := writer.WriteString(fmt.Sprintf("	%s: %s\n", c.Name, ""))
			if err != nil {
				continue
			}
		}
	}
}

func tableName(t string) string {
	if slices.Contains([]string{"steps", "shape"}, t) {
		return t + "_table"
	}
	return t
}

func typeName(t string) string {
	if strings.HasSuffix(t, "[]") {
		return strings.ReplaceAll(t, "[]", "")
	}
	return t
}
