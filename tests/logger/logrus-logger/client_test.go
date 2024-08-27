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
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../../cmd/mangosql/ --output ./client.go --package logruslogger --logger logrus ./schema.sql

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
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	user, err := db.User.Insert(UserCreate{
		Id:   1,
		Name: "tuna",
	})
	require.NoError(t, err)
	assert.Equal(t, "tuna", user.Name)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, "DB.User.Insert", entries[0].Message)
}

func TestInsertMany(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

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
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, "DB.User.InsertMany", entries[0].Message)
}

func TestUpdate(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	require.NoError(t, err)

	_, err = db.User.Update(UserUpdate{Id: 1, Name: "user1-updated"})
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, logrus.DebugLevel, entries[1].Level)
	assert.Equal(t, "DB.User.Update", entries[1].Message)
}

func TestUpsert(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
	require.NoError(t, err)

	_, err = db.User.Upsert(UserUpdate{Id: 1, Name: "user1-updated"})
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, logrus.DebugLevel, entries[1].Level)
	assert.Equal(t, "DB.User.Upsert", entries[1].Message)
}

func TestSoftDelete(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	require.NoError(t, err)

	err = db.User.DeleteSoft(2)
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, logrus.DebugLevel, entries[1].Level)
	assert.Equal(t, "DB.User.DeleteSoft", entries[1].Message)
}

func TestHardDelete(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	require.NoError(t, err)

	err = db.User.DeleteHard(2)
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, logrus.DebugLevel, entries[1].Level)
	assert.Equal(t, "DB.User.DeleteHard", entries[1].Message)
}

func TestFindMany(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.FindMany()
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, "DB.User.FindMany", entries[0].Message)
}

func TestCount(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.Count()
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, "DB.User.Count", entries[0].Message)
}

func TestFindManyError(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.User.FindMany(
		func(query SelectBuilder) SelectBuilder {
			return query.Where("unknownField = 'error'")
		},
	)
	require.Error(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.ErrorLevel, entries[0].Level)
	assert.Equal(t, "DB.User.FindMany", entries[0].Message)
}

func TestCustomQuery(t *testing.T) {
	db, closeDB, logs := newTestDB(t)
	defer closeDB()

	_, err := db.Queries.UserNotDeleted()
	require.NoError(t, err)

	entries := logs.AllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, "DB.Queries.UserNotDeleted", entries[0].Message)
}
