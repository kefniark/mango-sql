package bench

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	driver_sqlite "github.com/kefniark/mango-sql/tests/bench/sqlite"
	"github.com/stretchr/testify/assert"

	_ "modernc.org/sqlite"
)

func newBenchmarkDBSQLite(t *testing.B) (*driver_sqlite.DBClient, func()) {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}

	data, err := os.ReadFile("./schema.sqlite.sql")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(string(data))
	if err != nil {
		panic(err)
	}

	return driver_sqlite.New(db), func() {
		db.Close()
	}
}

func BenchmarkMangoSQLite(t *testing.B) {
	dbMangoSqlite, closeSqlite := newBenchmarkDBSQLite(t)
	defer closeSqlite()

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoSqlite.User.Insert(driver_sqlite.UserCreate{Id: uuid.NewString(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]driver_sqlite.UserCreate, value)
				for i := 0; i < len(create); i++ {
					create[i] = driver_sqlite.UserCreate{Id: uuid.NewString(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoSqlite.User.InsertMany(create)
				assert.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_sqlite.UserCreate, 10)
		for i := 0; i < len(create); i++ {
			create[i] = driver_sqlite.UserCreate{Id: uuid.NewString(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoSqlite.User.InsertMany(create)
		assert.NoError(t, err)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				user, err := dbMangoSqlite.User.FindById(users[i].Id)
				assert.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+fmt.Sprint(value), func(t *testing.B) {
			create := make([]driver_sqlite.UserCreate, value)
			ids := []string{}
			for i := 0; i < len(create); i++ {
				id := uuid.NewString()
				ids = append(ids, id)
				create[i] = driver_sqlite.UserCreate{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			_, err := dbMangoSqlite.User.InsertMany(create)
			assert.NoError(t, err)
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				entries, err := dbMangoSqlite.User.FindMany(
					dbMangoSqlite.User.Query.Id.In(ids...),
				)
				assert.NoError(t, err)
				assert.Equal(t, value, len(entries))
			}
		})
	}
}
