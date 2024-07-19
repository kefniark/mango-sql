package helpers

import (
	"embed"
	"testing"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
)

//go:embed migrations/*.sql
var sqlFS embed.FS

func NewDBConfig(t *testing.T) *pgtestdb.Config {
	t.Helper()

	gm := goosemigrator.New(
		"migrations",
		goosemigrator.WithFS(sqlFS),
		goosemigrator.WithTableName("goose_db_version"),
	)

	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}

	return pgtestdb.Custom(t, conf, gm)
}

func NewDBBenchConfig(t *testing.B) *pgtestdb.Config {
	t.Helper()

	gm := goosemigrator.New(
		"migrations",
		goosemigrator.WithFS(sqlFS),
		goosemigrator.WithTableName("goose_db_version"),
	)

	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}

	return pgtestdb.Custom(t, conf, gm)
}
