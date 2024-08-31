package parser

import (
	"os"
	"path"
	"testing"

	"github.com/kefniark/mango-sql/internal"
	"github.com/stretchr/testify/assert"
)

func TestMysql(t *testing.T) {
	folder := "./resources/mysql"
	entries, err := os.ReadDir(folder)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(path.Join(folder, entry.Name(), "schema.sql"))
			if err != nil {
				assert.NoError(t, err)
			}

			_, err = internal.ParseSchema(string(data))
			if err != nil {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPostgres(t *testing.T) {
	folder := "./resources/postgres"
	entries, err := os.ReadDir(folder)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(path.Join(folder, entry.Name(), "schema.sql"))
			if err != nil {
				assert.NoError(t, err)
			}

			_, err = internal.ParseSchema(string(data))
			if err != nil {
				assert.NoError(t, err)
			}
		})
	}
}
