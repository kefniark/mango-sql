package sqlite

import (
	"embed"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	data, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		t.Error(err)
	}

	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Error(err)
	}

	_, err = db.Exec(string(data))
	if err != nil {
		t.Error(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func TestInsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	val, err := db.Actor.Insert(ActorCreate{
		ActorId:    1,
		FirstName:  "tuna",
		LastName:   "fish",
		LastUpdate: time.Now(),
	})
	assert.NoError(t, err)
	assert.Equal(t, "tuna", val.FirstName)

	count, err := db.Actor.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	actor, err := db.Actor.FindById(1)
	assert.NoError(t, err)
	assert.Equal(t, "tuna", actor.FirstName)

	assert.EqualValues(t, val, actor)
}
