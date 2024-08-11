package types

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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
	db, err := pgxpool.New(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func TestNumeric(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	num, err := db.Numeric.Insert(NumericCreate{
		Smallserial: 1,
		Serial:      2,
		Bigserial:   3,
		Smallint:    4,
		Integer:     5,
		Bigint:      6,
		Numeric:     7.5,
		Float:       8.5,
	})
	assert.NoError(t, err)

	fmt.Println(num)
}

func TestText(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	value := "Hello world&-_+@"

	num, err := db.Text.Insert(TextCreate{
		Char1:    "a",
		Char2:    "b",
		Varchar1: value,
		Varchar2: value,
		Text:     value,
		Text2:    value,
	})
	assert.NoError(t, err)

	assert.Equal(t, "a", num.Char1)
	assert.Equal(t, "b", strings.TrimSpace(num.Char2))
	assert.Equal(t, value, num.Varchar1)
	assert.Equal(t, value, num.Varchar2)
	assert.Equal(t, value, num.Text)
	assert.Equal(t, value, num.Text2)
}

func TestArray(t *testing.T) {
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

func TestJSON(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	data := map[string]interface{}{"a": 1.0, "b": 2.5}

	val, err := db.Json.Insert(JsonCreate{
		Json1:  data,
		Jsonb1: data,
	})
	assert.NoError(t, err)
	assert.EqualValues(t, data, val.Json1)
	assert.EqualValues(t, data, val.Jsonb1)
}
