package composite

import (
	"embed"
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

	config := helpers.NewDBConfigWith(t, data, "postgres.composite")
	db, err := sqlx.Connect("postgres", config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func TestComposite(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	tag1, err := db.Tag.Insert(TagCreate{
		QuestionId: 1,
		TagId:      1,
		Name:       "Tuna",
	})
	assert.NoError(t, err)

	tag2, err := db.Tag.FindById(TagPrimaryKey{QuestionId: 1, TagId: 1})
	assert.NoError(t, err)

	assert.Equal(t, tag1.Name, tag2.Name)
}