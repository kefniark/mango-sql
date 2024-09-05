package bench

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type UserMaria struct {
	gorm.Model
	ID        uuid.UUID  `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func (UserMaria) TableName() string {
	return "users"
}

func (user *UserMaria) BeforeCreate(_ *gorm.DB) error {
	user.ID = uuid.New()
	return nil
}

func newBenchmarkDBGormMariaDB(t *testing.B) (*gorm.DB, func()) {
	t.Helper()
	id := gonanoid.MustGenerate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	testDB := "test_" + id

	dbAdmin, err := sqlx.Connect("mysql", "root:root@tcp(127.0.0.1:3306)/")
	dbAdmin.SetConnMaxIdleTime(time.Second * 10)
	require.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDB))
	require.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("GRANT ALL ON %s.* TO user@'%%' IDENTIFIED BY 'password' WITH GRANT OPTION;", testDB))
	require.NoError(t, err)

	_, err = dbAdmin.Exec("FLUSH PRIVILEGES;")
	require.NoError(t, err)

	db, err := sqlx.Connect("mysql", fmt.Sprintf("user:password@tcp(127.0.0.1:3306)/%s?parseTime=true&multiStatements=true", testDB))
	db.SetConnMaxIdleTime(time.Second * 10)
	require.NoError(t, err)

	fmt.Println("Create & Use DB", testDB)
	data, err := os.ReadFile("./schema.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(data))
	require.NoError(t, err)

	gormDB, err := gorm.Open(gormMysql.New(gormMysql.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return gormDB, func() {
		db.Close()
	}
}

func BenchmarkGormMariaDB(t *testing.B) {
	dbGormMaria, closeMaria := newBenchmarkDBGormMariaDB(t)
	defer closeMaria()

	t.Run("InsertOne", func(t *testing.B) {
		for range t.N {
			tx := dbGormMaria.Create(&User{Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, tx.Error)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]User, value)
				for i := range len(create) {
					create[i] = User{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				tx := dbGormMaria.Create(&create)
				require.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]User, 10)
		for i := range len(create) {
			create[i] = User{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormMaria.Create(&create)
		require.NoError(t, tx.Error)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				data := &User{}
				tx := dbGormMaria.Take(data, "id = ?", create[i].ID)
				require.NoError(t, tx.Error)

				assert.Equal(t, create[i].ID, data.ID)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			ids := []uuid.UUID{}
			for range value {
				user := &User{Name: "John Doe", Email: "john@email.com"}
				tx := dbGormMaria.Create(user)
				require.NoError(t, tx.Error)
				ids = append(ids, user.ID)
			}
			t.ResetTimer()

			for range t.N {
				data := []User{}
				tx := dbGormMaria.Find(&data, ids)
				require.NoError(t, tx.Error)
				assert.Len(t, data, value)
			}
		})
	}
}
