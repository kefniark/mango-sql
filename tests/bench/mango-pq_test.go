package bench

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	driver_pq "github.com/kefniark/mango-sql/tests/bench/pq"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBenchmarkDBPQ(t *testing.B) (*driver_pq.DBClient, func()) {
	t.Helper()
	config := helpers.NewDBBenchConfig(t)
	db, err := sqlx.Connect("postgres", config.URL())
	if err != nil {
		panic(err)
	}

	return driver_pq.New(db), func() {
		db.Close()
	}
}

func BenchmarkMangoPostgresPQ(t *testing.B) {
	dbMangoPq, closeDB := newBenchmarkDBPQ(t)
	defer closeDB()

	t.Run("InsertOne", func(t *testing.B) {
		for range t.N {
			_, err := dbMangoPq.User.Insert(driver_pq.UserCreate{Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]driver_pq.UserCreate, value)
				for i := range len(create) {
					create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoPq.User.InsertMany(create)
				require.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_pq.UserCreate, 10)
		for i := range len(create) {
			create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoPq.User.InsertMany(create)
		require.NoError(t, err)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				user, err := dbMangoPq.User.FindById(users[i].Id)
				require.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			create := make([]driver_pq.UserCreate, value)
			ids := []uuid.UUID{}
			for i := range len(create) {
				create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := dbMangoPq.User.InsertMany(create)
			for _, user := range users {
				ids = append(ids, user.Id)
			}
			require.NoError(t, err)
			t.ResetTimer()

			for range t.N {
				entries, err := dbMangoPq.User.FindMany(
					dbMangoPq.User.Query.Id.In(ids...),
				)
				require.NoError(t, err)
				assert.Len(t, entries, value)
			}
		})
	}
}
