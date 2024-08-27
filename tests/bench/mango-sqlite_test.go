package bench

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/jmoiron/sqlx"
	driver_sqlite "github.com/kefniark/mango-sql/tests/bench/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	id := int64(0)

	t.Run("InsertOne", func(t *testing.B) {
		for range t.N {
			id++
			_, err := dbMangoSqlite.User.Insert(driver_sqlite.UserCreate{Id: id, Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]driver_sqlite.UserCreate, value)
				for i := range len(create) {
					id++
					create[i] = driver_sqlite.UserCreate{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoSqlite.User.InsertMany(create)
				require.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_sqlite.UserCreate, 10)
		for i := range len(create) {
			id++
			create[i] = driver_sqlite.UserCreate{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoSqlite.User.InsertMany(create)
		require.NoError(t, err)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				user, err := dbMangoSqlite.User.FindById(users[i].Id)
				require.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			create := make([]driver_sqlite.UserCreate, value)
			ids := []int64{}
			for i := range len(create) {
				id++
				ids = append(ids, id)
				create[i] = driver_sqlite.UserCreate{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			_, err := dbMangoSqlite.User.InsertMany(create)
			require.NoError(t, err)
			t.ResetTimer()

			for range t.N {
				entries, err := dbMangoSqlite.User.FindMany(
					dbMangoSqlite.User.Query.Id.In(ids...),
				)
				require.NoError(t, err)
				assert.Len(t, entries, value)
			}
		})
	}
}
