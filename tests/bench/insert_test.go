package bench

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
	driver_pgx "github.com/kefniark/mango-sql/tests/bench/pgx"
	driver_pq "github.com/kefniark/mango-sql/tests/bench/pq"
	driver_sqlite "github.com/kefniark/mango-sql/tests/bench/sqlite"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"

	gormPostgres "gorm.io/driver/postgres"
	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
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

	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}
	return gormDB, func() {
		db.Close()
	}
}

func newBenchmarkDBGormSqlite(t *testing.B) (*gorm.DB, func()) {
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

	gormDB, err := gorm.Open(gormSqlite.New(gormSqlite.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(err)
	}
	return gormDB, func() {
		db.Close()
	}
}

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

type User struct {
	gorm.Model
	Id        uuid.UUID  `json:"id" db:"id" `
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func BenchmarkInsertOne(t *testing.B) {
	dbMangoPq, close := newBenchmarkDBPQ(t)
	defer close()
	dbMangoPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGormPgx, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()
	dbMangoSqlite, closeSqlite := newBenchmarkDBSQLite(t)
	defer closeSqlite()
	dbGormSqlite, closeGormSqlite := newBenchmarkDBGormSqlite(t)
	defer closeGormSqlite()

	t.Run("Insert One - Mango PQ", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoPq.User.Insert(driver_pq.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Insert One - Mango PGX", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoPgx.User.Insert(driver_pgx.UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Insert One - Gorm PGX", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			tx := dbGormPgx.Create(&User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})

	t.Run("Insert One - Mango Sqlite", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			_, err := dbMangoSqlite.User.Insert(driver_sqlite.UserCreate{Id: uuid.NewString(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Insert One - Gorm Sqlite", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			tx := dbGormSqlite.Create(&User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})
}

func BenchmarkInsertBulk(t *testing.B) {
	db, close := newBenchmarkDBPQ(t)
	defer close()
	dbPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()
	// dbSqlite, closeSqlite := newBenchmarkDBSQLite(t)
	// defer closeSqlite()

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

	// TODO: Fix Batch Insert
	// t.Run("Insert Bulk - Sqlite", func(t *testing.B) {
	// 	for i := 0; i < t.N; i++ {
	// 		create := make([]driver_sqlite.UserCreate, 50)
	// 		for i := 0; i < len(create); i++ {
	// 			create[i] = driver_sqlite.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
	// 		}

	// 		_, err := dbSqlite.User.InsertMany(create)
	// 		assert.NoError(t, err)
	// 	}
	// })
}

func BenchmarkSelect(t *testing.B) {
	db, close := newBenchmarkDBPQ(t)
	defer close()
	dbPgx, closePgx := newBenchmarkDBPGX(t)
	defer closePgx()
	dbGorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()
	// dbSqlite, closeSqlite := newBenchmarkDBSQLite(t)
	// defer closeSqlite()

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

	t.Run("Select - Mango PGX", func(t *testing.B) {
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

	t.Run("Select - Gorm PGX", func(t *testing.B) {
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

	// TODO: Fix Batch Insert
	// t.Run("Select - Sqlite", func(t *testing.B) {
	// 	create := make([]driver_sqlite.UserCreate, 10)
	// 	for i := 0; i < len(create); i++ {
	// 		create[i] = driver_sqlite.UserCreate{Id: uuid.NewString(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
	// 	}

	// 	users, err := dbSqlite.User.InsertMany(create)
	// 	assert.NoError(t, err)
	// 	t.ResetTimer()

	// 	for i := 0; i < t.N; i++ {
	// 		for i := 0; i < len(create); i++ {
	// 			user, err := dbSqlite.User.FindById(users[i].Id)
	// 			assert.NoError(t, err)
	// 			assert.Equal(t, users[i].Id, user.Id)
	// 		}
	// 	}
	// })
}
