package composite

import (
	"context"
	"embed"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../../cmd/mangosql/ --output client.go --package composite --logger console ./schema.sql

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	data, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		panic(err)
	}

	config := helpers.NewDBConfigWith(t, data, "postgres.composite")
	db, err := pgx.Connect(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close(context.Background())
	}
}

func TestComposite(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	tag1, err := db.Tag.Insert(TagCreate{
		QuestionId: 1,
		TagId:      1,
		Name:       "Tuna",
	})
	require.NoError(t, err)

	tag2, err := db.Tag.FindById(TagPrimaryKey{QuestionId: 1, TagId: 1})
	require.NoError(t, err)

	assert.Equal(t, tag1.Name, tag2.Name)
}
