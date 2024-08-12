package bench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	driver_pgx "github.com/kefniark/mango-sql/tests/bench/pgx"
	driver_pq "github.com/kefniark/mango-sql/tests/bench/pq"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

func newBenchmarkDBGorm(t *testing.B) (*gorm.DB, func()) {
	t.Helper()
	config := helpers.NewDBBenchConfig(t)
	db, err := sqlx.Connect("pgx", config.URL())
	if err != nil {
		panic(err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return gormDB, func() {
		db.Close()
	}
}

type User struct {
	gorm.Model
	Id        uuid.UUID  `json:"id" db:"id" `
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func benchmarkInsertOne(t *testing.B) {
	dbPq, close := newBenchmarkDBPQ(t)
	defer close()
	dbPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()

	t.Run("Insert One - Mango PQ", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbPq.User.Insert(driver_pq.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Insert One - Mango PGX", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbPgx.User.Insert(driver_pgx.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Insert One - Gorm", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			tx := dbGorm.Create(&User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})
}

func benchmarkInsertBulk(t *testing.B) {
	db, close := newBenchmarkDBPQ(t)
	defer close()
	dbPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()

	t.Run("Insert Bulk - Mango PQ", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			create := make([]driver_pq.UserCreate, 50)
			for i := 0; i < len(create); i++ {
				create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			_, err := db.User.InsertMany(create)
			assert.NoError(t, err)
		}
	})

	t.Run("Insert Bulk - Mango PGX", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			create := make([]driver_pgx.UserCreate, 50)
			for i := 0; i < len(create); i++ {
				create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			_, err := dbPgx.User.InsertMany(create)
			assert.NoError(t, err)
		}
	})

	t.Run("Insert Bulk - Gorm", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			create := make([]User, 50)
			for i := 0; i < len(create); i++ {
				create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			tx := dbGorm.Create(&create)
			assert.NoError(t, tx.Error)
		}
	})
}

func benchmarkSelect(t *testing.B) {
	db, close := newBenchmarkDBPQ(t)
	defer close()
	dbPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()

	t.Run("Select - Mango PQ", func(t *testing.B) {
		create := make([]driver_pq.UserCreate, 10)
		for i := 0; i < len(create); i++ {
			create[i] = driver_pq.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := db.User.InsertMany(create)
		assert.NoError(t, err)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				user, err := db.User.FindById(users[i].Id)
				assert.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	t.Run("Select - Mango Pgx", func(t *testing.B) {
		create := make([]driver_pgx.UserCreate, 10)
		for i := 0; i < len(create); i++ {
			create[i] = driver_pgx.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbPgx.User.InsertMany(create)
		assert.NoError(t, err)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				user, err := dbPgx.User.FindById(users[i].Id)
				assert.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	t.Run("Select - Gorm", func(t *testing.B) {
		create := make([]User, 10)
		for i := 0; i < len(create); i++ {
			create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGorm.Create(&create)
		assert.NoError(t, tx.Error)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				data := &User{}
				tx := dbGorm.Take(data, "id = ?", create[i].Id)
				assert.NoError(t, tx.Error)

				assert.Equal(t, create[i].Id, data.Id)
			}
		}
	})
}

func Benchmark(t *testing.B) {
	benchmarkInsertOne(t)
	benchmarkInsertBulk(t)
	benchmarkSelect(t)
}
