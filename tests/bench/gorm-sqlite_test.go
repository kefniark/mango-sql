package bench

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

type UserSqlite struct {
	gorm.Model
	Id        int        `json:"id" db:"id" `
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func (UserSqlite) TableName() string {
	return "users"
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

func BenchmarkGormSQLite(t *testing.B) {
	dbGormSqlite, closeGormSqlite := newBenchmarkDBGormSqlite(t)
	defer closeGormSqlite()

	id := 0

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			id++
			tx := dbGormSqlite.Create(&UserSqlite{Id: id, Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]UserSqlite, value)
				for i := 0; i < len(create); i++ {
					id++
					create[i] = UserSqlite{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				tx := dbGormSqlite.Create(&create)
				assert.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]UserSqlite, 10)
		for i := 0; i < len(create); i++ {
			id++
			create[i] = UserSqlite{Id: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormSqlite.Create(&create)
		assert.NoError(t, tx.Error)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				data := &UserSqlite{}
				tx := dbGormSqlite.Take(data, "id = ?", create[i].Id)
				assert.NoError(t, tx.Error)

				assert.Equal(t, create[i].Id, data.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+fmt.Sprint(value), func(t *testing.B) {
			create := make([]UserSqlite, value)
			ids := []int{}
			for i := 0; i < len(create); i++ {
				id++
				user := &UserSqlite{Id: id, Name: "John Doe", Email: "john@email.com"}
				tx := dbGormSqlite.Create(user)
				assert.NoError(t, tx.Error)
				ids = append(ids, user.Id)
			}
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				data := []UserSqlite{}
				tx := dbGormSqlite.Find(&data, ids)
				assert.NoError(t, tx.Error)
				assert.Equal(t, value, len(data))
			}
		})
	}
}
