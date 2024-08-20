package bench

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	driver_pq "github.com/kefniark/mango-sql/tests/bench/pq"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
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
	dbMangoPq, close := newBenchmarkDBPQ(t)
	defer close()

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoPq.User.Insert(driver_pq.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]driver_pq.UserCreate, value)
				for i := 0; i < len(create); i++ {
					create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoPq.User.InsertMany(create)
				assert.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_pq.UserCreate, 10)
		for i := 0; i < len(create); i++ {
			create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoPq.User.InsertMany(create)
		assert.NoError(t, err)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				user, err := dbMangoPq.User.FindById(users[i].Id)
				assert.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+fmt.Sprint(value), func(t *testing.B) {
			create := make([]driver_pq.UserCreate, value)
			ids := []uuid.UUID{}
			for i := 0; i < len(create); i++ {
				create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := dbMangoPq.User.InsertMany(create)
			for _, user := range users {
				ids = append(ids, user.Id)
			}
			assert.NoError(t, err)
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				entries, err := dbMangoPq.User.FindMany(
					dbMangoPq.User.Query.Id.In(ids...),
				)
				assert.NoError(t, err)
				assert.Equal(t, value, len(entries))
			}
		})
	}
}
