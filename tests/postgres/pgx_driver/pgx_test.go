package main

import (
	"context"
	"embed"
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

	config := helpers.NewDBConfigWith(t, data, "postgres")
	db, err := pgxpool.New(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func TestDBInsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user1, err1 := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err1)

	user1Check, err := db.User.FindById(user1.Id)
	assert.NoError(t, err)
	assert.Equal(t, user1.Name, user1Check.Name)
}
