package types

import (
	"embed"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
)

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	data, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		panic(err)
	}

	config := helpers.NewDBConfigWith(t, data, "postgres.types")
	db, err := sqlx.Connect("postgres", config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func TestArrayFilters(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	list, err := db.List.Insert(ListCreate{
		Integer1:  []int64{1, 2, 3},
		Integer2:  &[]int64{4, 5, 6},
		Smallint1: []int64{1, 2, 3},
		Smallint2: &[]int64{4, 5, 6},
		Bigint1:   []int64{1, 2, 3},
		Bigint2:   &[]int64{4, 5, 6},
		Text1:     []string{"a", "b", "c"},
		Text2:     &[]string{"d", "e", "f"},
	})
	assert.NoError(t, err)

	fmt.Println(list)
}
