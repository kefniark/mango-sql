package bench

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gormSqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

type UserSqlite struct {
	gorm.Model
	ID        int        `json:"id" db:"id" `
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
		for range t.N {
			id++
			tx := dbGormSqlite.Create(&UserSqlite{ID: id, Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, tx.Error)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]UserSqlite, value)
				for i := range len(create) {
					id++
					create[i] = UserSqlite{ID: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				tx := dbGormSqlite.Create(&create)
				require.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]UserSqlite, 10)
		for i := range len(create) {
			id++
			create[i] = UserSqlite{ID: id, Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormSqlite.Create(&create)
		require.NoError(t, tx.Error)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				data := &UserSqlite{}
				tx := dbGormSqlite.Take(data, "id = ?", create[i].ID)
				require.NoError(t, tx.Error)

				assert.Equal(t, create[i].ID, data.ID)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			ids := []int{}
			for range value {
				id++
				user := &UserSqlite{ID: id, Name: "John Doe", Email: "john@email.com"}
				tx := dbGormSqlite.Create(user)
				require.NoError(t, tx.Error)
				ids = append(ids, user.ID)
			}
			t.ResetTimer()

			for range t.N {
				data := []UserSqlite{}
				tx := dbGormSqlite.Find(&data, ids)
				require.NoError(t, tx.Error)
				assert.Len(t, data, value)
			}
		})
	}
}
