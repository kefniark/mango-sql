package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func newBenchmarkDB(t *testing.B) (*DBClient, func()) {
	t.Helper()
	config := helpers.NewDBBenchConfig(t)
	db, err := sqlx.Connect("postgres", config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
	}
}

func newTestDB(t *testing.T) (*DBClient, func()) {
	config := helpers.NewDBConfig(t)
	db, err := sqlx.Connect("postgres", config.URL())
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

	user1, err1 := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err1)

	user1Check, err := db.User.GetById(user1.Id)
	assert.NoError(t, err)
	assert.Equal(t, user1.Name, user1Check.Name)
}

func TestDBInsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	ids, err := db.User.CreateMany([]UserCreate{
		{Name: "user2", Email: "user2@email.com"},
		{Name: "user3", Email: "user3@email.com"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids))
}

func TestDBUpdate(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)

	_, err = db.User.Update(UserUpdate{Id: user.Id, Email: user.Email, Name: "user1-updated"})
	assert.NoError(t, err)
}

func TestDBUpdateMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	ids, err := db.User.CreateMany([]UserCreate{
		{Name: "user1", Email: "user1@email.com"},
		{Name: "user2", Email: "user2@email.com"},
	})
	assert.NoError(t, err)

	_, err = db.User.UpdateMany([]UserUpdate{
		{Id: ids[0], Email: "user1@email.com", Name: "user1-updated"},
		{Id: ids[1], Email: "user2@email.com", Name: "user2-updated"},
	})
	assert.NoError(t, err)

	user, err := db.User.GetById(ids[0])
	assert.NoError(t, err)

	assert.Equal(t, "user1-updated", user.Name)
}

func TestDBUpsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.All(0, 1)
	assert.NoError(t, err)

	user1, err := db.User.Upsert(UserUpdate{Email: "usernew@localhost", Name: "usernew"})
	assert.NoError(t, err)

	user2, err := db.User.Upsert(UserUpdate{Id: users[0].Id, Email: "user1-updated", Name: "user1-updated"})
	assert.NoError(t, err)

	all, err := db.User.Count()
	assert.NoError(t, err)

	user1Check, err := db.User.GetById(user1.Id)
	assert.NoError(t, err)

	user2Check, err := db.User.GetById(user2.Id)
	assert.NoError(t, err)

	assert.Equal(t, 5, all)
	assert.Equal(t, user2.Id, users[0].Id)
	assert.Equal(t, user1.Name, user1Check.Name)
	assert.Equal(t, user2.Name, user2Check.Name)
}

func TestDBUpsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.All(0, 1)
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

// func TestDBSave(t *testing.T) {
// 	db, close := newDB(t)
// 	defer close()

// 	user, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
// 	assert.NoError(t, err)

// 	originalUpdate := user.UpdatedAt
// 	user.Name = "user1-up"
// 	err = user.Save()

// 	assert.NoError(t, err)

// 	user2, err := db.User.GetById(user.Id)
// 	assert.NoError(t, err)

// 	assert.Equal(t, "user1-up", user2.Name)
// 	assert.NotEqual(t, originalUpdate, user.UpdatedAt)
// }

func TestDBWhere(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.Where(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name ILIKE $1 OR name ILIKE $2", "%user1%", "%user2%")
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))
}

func TestDBFind(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.Where(func(cond SelectBuilder) SelectBuilder {
		return cond.Offset(0).Limit(10)
	})

	assert.NoError(t, err)
	assert.Equal(t, 4, len(users))
}

func TestDBCount(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	count, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	count, err = db.User.CountWhere(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name = ?", "user1")
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDBSoftDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user1, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)
	// user2, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	// assert.NoError(t, err)

	assert.NoError(t, db.User.DeleteSoft(user1.Id))

	user1Check, err := db.User.GetById(user1.Id)
	assert.NoError(t, err)
	// user2Check, err := db.User.GetById(user2.Id)
	// assert.NoError(t, err)

	assert.NotNil(t, user1Check.DeletedAt)
	// assert.NotNil(t, user2Check.DeletedAt)
}

func TestDBHardDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user1, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)
	// user2, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@email.com"})
	// assert.NoError(t, err)

	assert.NoError(t, db.User.DeleteHard(user1.Id))
	// assert.NoError(t, user2.DeleteHard())

	_, err = db.User.GetById(user1.Id)
	assert.Error(t, err)

	// _, err = db.User.GetById(user2.Id)
	// assert.Error(t, err)
}

func TestTransactionInsertCommit(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	err := db.Transaction(func(tx *DBClient) error {
		_, err1 := tx.User.Create(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, err2 := tx.User.Create(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, err3 := tx.User.Create(UserCreate{Name: "user3", Email: "user3@localhost"})
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
		_, _ = tx.User.Create(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, _ = tx.User.Create(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, _ = tx.User.Create(UserCreate{Name: "user3", Email: "user3@localhost"})
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
		_, _ = tx.User.Create(UserCreate{Name: "user1", Email: "user1@localhost"})
		_, _ = tx.User.Create(UserCreate{Name: "user2", Email: "user2@localhost"})
		_, _ = tx.User.Create(UserCreate{Name: "user3", Email: "user3@localhost"})

		panic("rollback")
	})
	assert.Error(t, err)

	val, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, val)
}

func BenchmarkInsert(t *testing.B) {
	sizes := []int{1, 50, 500, 5000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Create %d users (single insert)", size), func(t *testing.B) {
			db, close := newBenchmarkDB(t)
			defer close()

			for i := 0; i < size; i++ {
				_, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@localhost"})
				if err != nil {
					assert.NoError(t, err)
				}
			}

			val, err := db.User.Count()
			assert.NoError(t, err)
			assert.Equal(t, size+4, val)
		})

		t.Run(fmt.Sprintf("Create %d users (inserts in transaction)", size), func(t *testing.B) {
			db, close := newBenchmarkDB(t)
			defer close()

			err := db.Transaction(func(tx *DBClient) error {
				for i := 0; i < size; i++ {
					_, err := db.User.Create(UserCreate{Name: "user1", Email: "user1@localhost"})
					if err != nil {
						return err
					}
				}
				return nil
			})

			assert.NoError(t, err)
			val, err := db.User.Count()
			assert.NoError(t, err)
			assert.Equal(t, size+4, val)
		})

		t.Run(fmt.Sprintf("Create %d users (batch inserts)", size), func(t *testing.B) {
			db, close := newBenchmarkDB(t)
			defer close()

			users := make([]UserCreate, size)
			for i := 0; i < size; i++ {
				users[i] = UserCreate{Name: "user1", Email: "user1@localhost"}
			}

			_, err := db.User.CreateMany(users)
			assert.NoError(t, err)

			val, err := db.User.Count()
			assert.NoError(t, err)
			assert.Equal(t, size+4, val)
		})

		t.Run(fmt.Sprintf("Create %d users (batch inserts in transaction)", size), func(t *testing.B) {
			db, close := newBenchmarkDB(t)
			defer close()

			users := make([]UserCreate, size)
			for i := 0; i < size; i++ {
				users[i] = UserCreate{Name: "user1", Email: "user1@localhost"}
			}

			err := db.Transaction(func(tx *DBClient) error {
				_, err := db.User.CreateMany(users)
				return err
			})
			assert.NoError(t, err)

			val, err := db.User.Count()
			assert.NoError(t, err)
			assert.Equal(t, size+4, val)
		})
	}
}
