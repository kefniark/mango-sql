package bench

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newBenchmarkDB(t *testing.B) (*DBClient, func()) {
	t.Helper()
	config := helpers.NewDBBenchConfig(t)
	db, err := sqlx.Connect("pgx", config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close()
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

func BenchmarkInsert(t *testing.B) {
	size := 10

	db, close := newBenchmarkDB(t)
	defer close()
	gorm, closeGorm := newBenchmarkDBGorm(t)
	defer closeGorm()

	t.Run("Mango Insert", func(t *testing.B) {
		for i := 0; i < size; i++ {
			_, err := db.User.Insert(UserCreate{Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, err)
		}
	})

	t.Run("Gorm Insert", func(t *testing.B) {
		for i := 0; i < size; i++ {
			tx := gorm.Create(&User{Id: uuid.New(), Name: "John Doe", Email: "john@email.com"})
			assert.NoError(t, tx.Error)
		}
	})

	t.Run("Mango Insert Bulk", func(t *testing.B) {
		for i := 0; i < size; i++ {
			create := make([]UserCreate, 50)
			for i := 0; i < len(create); i++ {
				create[i] = UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			_, err := db.User.InsertMany(create)
			assert.NoError(t, err)
		}
	})

	t.Run("Gorm Insert Bulk", func(t *testing.B) {
		for i := 0; i < size; i++ {
			create := make([]User, 50)
			for i := 0; i < len(create); i++ {
				create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			tx := gorm.Create(&create)
			assert.NoError(t, tx.Error)
		}
	})

	t.Run("Mango Select", func(t *testing.B) {
		for i := 0; i < size; i++ {
			create := make([]UserCreate, 10)
			for i := 0; i < len(create); i++ {
				create[i] = UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := db.User.InsertMany(create)
			assert.NoError(t, err)

			for i := 0; i < len(create); i++ {
				user, err := db.User.FindById(users[i])
				assert.NoError(t, err)
				assert.Equal(t, users[i], user.Id)
			}
		}
	})

	t.Run("Gorm Select", func(t *testing.B) {
		for i := 0; i < size; i++ {
			create := make([]User, 10)
			for i := 0; i < len(create); i++ {
				create[i] = User{Id: uuid.New(), Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			tx := gorm.Create(&create)
			assert.NoError(t, tx.Error)

			for i := 0; i < len(create); i++ {
				data := &User{}
				tx := gorm.Take(data, "id = ?", create[i].Id)
				assert.NoError(t, tx.Error)

				assert.Equal(t, create[i].Id, data.Id)
			}
		}
	})
}
