package bench

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	gorm.Model
	ID        uuid.UUID  `json:"id" db:"id"`
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

func (user *User) BeforeCreate(_ *gorm.DB) error {
	user.ID = uuid.New()
	return nil
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
		for range t.N {
			tx := dbGormPgx.Create(&User{Name: "John Doe", Email: "john@email.com"})
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

				tx := dbGormPgx.Create(&create)
				require.NoError(t, tx.Error)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]User, 10)
		for i := range len(create) {
			create[i] = User{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		tx := dbGormPgx.Create(&create)
		require.NoError(t, tx.Error)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				data := &User{}
				tx := dbGormPgx.Take(data, "id = ?", create[i].ID)
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
				tx := dbGormPgx.Create(user)
				require.NoError(t, tx.Error)
				ids = append(ids, user.ID)
			}
			t.ResetTimer()

			for range t.N {
				data := []User{}
				tx := dbGormPgx.Find(&data, ids)
				require.NoError(t, tx.Error)
				assert.Len(t, data, value)
			}
		})
	}
}
