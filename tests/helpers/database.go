package helpers

import (
	"embed"
	"testing"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/goosemigrator"
)

//go:embed migrations/*.sql
var sqlFS embed.FS

func NewDBConfigWith(t *testing.T, data []byte, _ string) *pgtestdb.Config {
	t.Helper()

	var migrator pgtestdb.Migrator = &SchemaMigrator{
		Data: data,
	}

	conf := pgtestdb.Config{
		DriverName: "postgres",
		User:       "postgres",
		Password:   "password",
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}

	return pgtestdb.Custom(t, conf, migrator)
}

func NewDBConfig(t *testing.T, fs embed.FS) *pgtestdb.Config {
	t.Helper()

	gm := goosemigrator.New(
		"migrations",
		goosemigrator.WithFS(fs),
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
