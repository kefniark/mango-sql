package bench

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

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

func BenchmarkGormSQLite(t *testing.B) {
	dbGormSqlite, closeGormSqlite := newBenchmarkDBGormSqlite(t)
	defer closeGormSqlite()

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			tx := dbGormSqlite.Create(&User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]User, value)
				for i := 0; i < len(create); i++ {
					create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				tx := dbGormSqlite.Create(&create)
				assert.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]User, 10)
		for i := 0; i < len(create); i++ {
			create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormSqlite.Create(&create)
		assert.NoError(t, tx.Error)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				data := &User{}
				tx := dbGormSqlite.Take(data, "id = ?", create[i].Id)
				assert.NoError(t, tx.Error)

				assert.Equal(t, create[i].Id, data.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+fmt.Sprint(value), func(t *testing.B) {
			create := make([]User, value)
			ids := []uuid.UUID{}
			for i := 0; i < len(create); i++ {
				user := &User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"}
				tx := dbGormSqlite.Create(user)
				assert.NoError(t, tx.Error)
				ids = append(ids, user.Id)
			}
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				data := []User{}
				tx := dbGormSqlite.Find(&data, ids)
				assert.NoError(t, tx.Error)
				assert.Equal(t, value, len(data))
			}
		})
	}
}
