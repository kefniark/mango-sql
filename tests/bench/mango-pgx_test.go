package bench

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	driver_pgx "github.com/kefniark/mango-sql/tests/bench/pgx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		for range t.N {
			_, err := dbMangoPgx.User.Insert(driver_pgx.UserCreate{Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]driver_pgx.UserCreate, value)
				for i := range len(create) {
					create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoPgx.User.InsertMany(create)
				require.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_pgx.UserCreate, 10)
		for i := range len(create) {
			create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoPgx.User.InsertMany(create)
		require.NoError(t, err)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				user, err := dbMangoPgx.User.FindById(users[i].Id)
				require.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			create := make([]driver_pgx.UserCreate, value)
			ids := []uuid.UUID{}
			for i := range len(create) {
				create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := dbMangoPgx.User.InsertMany(create)
			for _, user := range users {
				ids = append(ids, user.Id)
			}
			require.NoError(t, err)
			t.ResetTimer()

			for range t.N {
				entries, err := dbMangoPgx.User.FindMany(
					dbMangoPgx.User.Query.Id.In(ids...),
				)
				require.NoError(t, err)
				assert.Len(t, entries, value)
			}
		})
	}
}
