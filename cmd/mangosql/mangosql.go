package main

import (
	"fmt"
	"os"

	"github.com/kefniark/mango-sql/cmd/mangosql/actions/codegen"
	"github.com/kefniark/mango-sql/cmd/mangosql/actions/diagram"
	"github.com/urfave/cli/v2"
)

const (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := &cli.App{
		Version:  fmt.Sprintf("%s (%s - %s)", version, commit, date),
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
			&cli.BoolFlag{
				Name:    "inline",
				Aliases: []string{"i"},
				Usage:   "Output to console",
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
		Action: codegen.Action,
		Commands: []*cli.Command{
			diagram.Command(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("\033[31mError\033[0m:", err)
		os.Exit(1)
	}
}
