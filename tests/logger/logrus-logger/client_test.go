package logruslogger

import (
	"context"
	"embed"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

//go:generate go run ../../../cmd/mangosql/ --package logruslogger --logger logrus ./schema.sql

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func(), *test.Hook) {
	data, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		panic(err)
	}

	config := helpers.NewDBConfigWith(t, data, "postgres.logrus-logger")
	db, err := pgx.Connect(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	logger, logHook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	return New(db, logger), func() {
		db.Close(context.Background())
	}, logHook
}

func TestInsert(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	user, err := db.User.Insert(UserCreate{
		Id:   1,
		Name: "tuna",
	})
	assert.NoError(t, err)
	assert.Equal(t, "tuna", user.Name)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.DebugLevel)
	assert.Equal(t, entries[0].Message, "DB.User.Insert")
}

func TestInsertMany(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.InsertMany([]UserCreate{
		{
			Id:   1,
			Name: "tuna",
		},
		{
			Id:   2,
			Name: "salmon",
		},
	})
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.DebugLevel)
	assert.Equal(t, entries[0].Message, "DB.User.InsertMany")
}

func TestUpdate(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	assert.NoError(t, err)

	_, err = db.User.Update(UserUpdate{Id: 1, Name: "user1-updated"})
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, entries[1].Level, logrus.DebugLevel)
	assert.Equal(t, entries[1].Message, "DB.User.Update")
}

func TestUpsert(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
	assert.NoError(t, err)

	_, err = db.User.Upsert(UserUpdate{Id: 1, Name: "user1-updated"})
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, entries[1].Level, logrus.DebugLevel)
	assert.Equal(t, entries[1].Message, "DB.User.Upsert")
}

func TestSoftDelete(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	assert.NoError(t, err)

	err = db.User.DeleteSoft(2)
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, entries[1].Level, logrus.DebugLevel)
	assert.Equal(t, entries[1].Message, "DB.User.DeleteSoft")
}

func TestHardDelete(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	assert.NoError(t, err)

	err = db.User.DeleteHard(2)
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 2, len(entries))
	assert.Equal(t, entries[1].Level, logrus.DebugLevel)
	assert.Equal(t, entries[1].Message, "DB.User.DeleteHard")
}

func TestFindMany(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.FindMany()
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.DebugLevel)
	assert.Equal(t, entries[0].Message, "DB.User.FindMany")
}

func TestCount(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.Count()
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.DebugLevel)
	assert.Equal(t, entries[0].Message, "DB.User.Count")
}

func TestFindManyError(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.User.FindMany(
		func(query SelectBuilder) SelectBuilder {
			return query.Where("unknownField = 'error'")
		},
	)
	assert.Error(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.ErrorLevel)
	assert.Equal(t, entries[0].Message, "DB.User.FindMany")
}

func TestCustomQuery(t *testing.T) {
	db, close, logs := newTestDB(t)
	defer close()

	_, err := db.Queries.UserNotDeleted()
	assert.NoError(t, err)

	entries := logs.AllEntries()
	assert.Equal(t, 1, len(entries))
	assert.Equal(t, entries[0].Level, logrus.DebugLevel)
	assert.Equal(t, entries[0].Message, "DB.Queries.UserNotDeleted")
}
