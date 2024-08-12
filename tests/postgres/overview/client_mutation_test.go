package overview

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
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
	db, err := pgx.Connect(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close(context.Background())
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

func TestDBInsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	ids, err := db.User.InsertMany([]UserCreate{
		{Name: "user2", Email: "user2@email.com"},
		{Name: "user3", Email: "user3@email.com"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids))
}

func TestDBUpdate(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)

	_, err = db.User.Update(UserUpdate{Id: user.Id, Email: user.Email, Name: "user1-updated"})
	assert.NoError(t, err)
}

func TestDBUpdateMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	ids, err := db.User.InsertMany([]UserCreate{
		{Name: "user1", Email: "user1@email.com"},
		{Name: "user2", Email: "user2@email.com"},
	})
	assert.NoError(t, err)

	_, err = db.User.UpdateMany([]UserUpdate{
		{Id: ids[0].Id, Email: "user1@email.com", Name: "user1-updated"},
		{Id: ids[1].Id, Email: "user2@email.com", Name: "user2-updated"},
	})
	assert.NoError(t, err)

	user, err := db.User.FindById(ids[0].Id)
	assert.NoError(t, err)

	assert.Equal(t, "user1-updated", user.Name)
}

func TestDBUpsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.FindMany()
	assert.NoError(t, err)

	user1, err := db.User.Upsert(UserUpdate{Email: "usernew@localhost", Name: "usernew"})
	assert.NoError(t, err)

	user2, err := db.User.Upsert(UserUpdate{Id: users[0].Id, Email: "user1-updated", Name: "user1-updated"})
	assert.NoError(t, err)

	all, err := db.User.Count()
	assert.NoError(t, err)

	usersCheck, err := db.User.FindByIds(user1.Id, user2.Id)
	assert.NoError(t, err)

	assert.Equal(t, 5, all)
	assert.Equal(t, user2.Id, users[0].Id)
	assert.Equal(t, user1.Name, usersCheck[0].Name)
	assert.Equal(t, user2.Name, usersCheck[1].Name)
}

func TestDBUpsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.FindMany()
	assert.NoError(t, err)

	ids, err := db.User.UpsertMany([]UserUpdate{
		{Email: "usernew@localhost", Name: "usernew"},
		{Id: users[0].Id, Email: "user1-updated", Name: "user1-updated"},
	})
	assert.NoError(t, err)

	all, err := db.User.Count()
	assert.NoError(t, err)

	assert.Equal(t, 5, all)
	assert.Equal(t, 2, len(ids))
}

func TestDBSoftDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user1, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)
	// user2, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	// assert.NoError(t, err)

	assert.NoError(t, db.User.DeleteSoft(user1.Id))

	user1Check, err := db.User.FindById(user1.Id)
	assert.NoError(t, err)
	// user2Check, err := db.User.GetById(user2.Id)
	// assert.NoError(t, err)

	assert.NotNil(t, user1Check.DeletedAt)
	// assert.NotNil(t, user2Check.DeletedAt)
}

func TestDBHardDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user1, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)
	// user2, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	// assert.NoError(t, err)

	assert.NoError(t, db.User.DeleteHard(user1.Id))
	// assert.NoError(t, user2.DeleteHard())

	data, err := db.User.FindById(user1.Id)
	fmt.Println(data, err)
	assert.Error(t, err)

	// _, err = db.User.GetById(user2.Id)
	// assert.Error(t, err)
}

func TestTransactionInsertCommit(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	err := db.Transaction(func(tx *DBClient) error {
		_, err1 := tx.User.Insert(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, err2 := tx.User.Insert(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, err3 := tx.User.Insert(UserCreate{Name: "user3", Email: "user3@localhost"})
		return errors.Join(err1, err2, err3)
	})
	assert.NoError(t, err)

	val, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 7, val)
}

func TestTransactionInsertRollback(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	err := db.Transaction(func(tx *DBClient) error {
		_, _ = tx.User.Insert(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, _ = tx.User.Insert(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, _ = tx.User.Insert(UserCreate{Name: "user3", Email: "user3@localhost"})
		return fmt.Errorf("rollback")
	})
	assert.Error(t, err)

	val, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, val)
}

func TestTransactionInsertRollbackPanic(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	err := db.Transaction(func(tx *DBClient) error {
		_, _ = tx.User.Insert(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, _ = tx.User.Insert(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, _ = tx.User.Insert(UserCreate{Name: "user3", Email: "user3@localhost"})

		panic("rollback")
	})
	assert.Error(t, err)

	val, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, val)
}
