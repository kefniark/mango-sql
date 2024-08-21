package bench

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"

	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	gorm.Model
	Id        uuid.UUID  `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	user.Id = uuid.New()
	return
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

func BenchmarkGormPostgresPGX(t *testing.B) {
	dbGormPgx, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()

	t.Run("InsertOne", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			tx := dbGormPgx.Create(&User{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+fmt.Sprint(value), func(t *testing.B) {
			for i := 0; i < t.N; i++ {
				create := make([]User, value)
				for i := 0; i < len(create); i++ {
					create[i] = User{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				tx := dbGormPgx.Create(&create)
				assert.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]User, 10)
		for i := 0; i < len(create); i++ {
			create[i] = User{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormPgx.Create(&create)
		assert.NoError(t, tx.Error)
		t.ResetTimer()

		for i := 0; i < t.N; i++ {
			for i := 0; i < len(create); i++ {
				data := &User{}
				tx := dbGormPgx.Take(data, "id = ?", create[i].Id)
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
				user := &User{Name: "John Doe", Email: "john@email.com"}
				tx := dbGormPgx.Create(user)
				assert.NoError(t, tx.Error)
				ids = append(ids, user.Id)
			}
			t.ResetTimer()

			for i := 0; i < t.N; i++ {
				data := []User{}
				tx := dbGormPgx.Find(&data, ids)
				assert.NoError(t, tx.Error)
				assert.Equal(t, value, len(data))
			}
		})
	}
}
