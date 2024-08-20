package bench

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	driver_pgx "github.com/kefniark/mango-sql/tests/bench/pgx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func newBenchmarkDBPGX(t *testing.B) (*driver_pgx.DBClient, func()) {
	t.Helper()
	config := helpers.NewDBBenchConfig(t)
	db, err := pgx.Connect(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return driver_pgx.New(db), func() {
		db.Close(context.Background())
	}
}

func BenchmarkMangoPostgresPGX(t *testing.B) {
	dbMangoPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoPgx.User.Insert(driver_pgx.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]driver_pgx.UserCreate, value)
				for i := 0; i < len(create); i++ {
					create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoPgx.User.InsertMany(create)
				assert.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_pgx.UserCreate, 10)
		for i := 0; i < len(create); i++ {
			create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoPgx.User.InsertMany(create)
		assert.NoError(t, err)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				user, err := dbMangoPgx.User.FindById(users[i].Id)
				assert.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+fmt.Sprint(value), func(t *testing.B) {
			create := make([]driver_pgx.UserCreate, value)
			ids := []uuid.UUID{}
			for i := 0; i < len(create); i++ {
				create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := dbMangoPgx.User.InsertMany(create)
			for _, user := range users {
				ids = append(ids, user.Id)
			}
			assert.NoError(t, err)
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				entries, err := dbMangoPgx.User.FindMany(
					dbMangoPgx.User.Query.Id.In(ids...),
				)
				assert.NoError(t, err)
				assert.Equal(t, value, len(entries))
			}
		})
	}
}
