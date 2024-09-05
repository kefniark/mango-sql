package bench

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	driver_mariadb "github.com/kefniark/mango-sql/tests/bench/mariadb"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
)

func newBenchmarkMariaDB(t *testing.B) (*driver_mariadb.DBClient, func()) {
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

	return driver_mariadb.New(db), func() {
		dbAdmin.MustExec(fmt.Sprintf("DROP DATABASE %s;", testDB))
		fmt.Println("Cleanup DB", testDB)
		dbAdmin.Close()
		db.Close()
	}
}

func BenchmarkMangoMariaDB(t *testing.B) {
	dbMangoMariaDB, closeMariaDB := newBenchmarkMariaDB(t)
	defer closeMariaDB()

	t.Run("InsertOne", func(t *testing.B) {
		for range t.N {
			_, err := dbMangoMariaDB.User.Insert(driver_mariadb.UserCreate{Name: "John Doe", Email: "john@email.com"})
			require.NoError(t, err)
		}
	})

	for _, value := range samples {
		t.Run("InsertMany_"+strconv.Itoa(value), func(t *testing.B) {
			for range t.N {
				create := make([]driver_mariadb.UserCreate, value)
				for i := range len(create) {
					create[i] = driver_mariadb.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
				}

				_, err := dbMangoMariaDB.User.InsertMany(create)
				require.NoError(t, err)
			}
		})
	}

	t.Run("FindById", func(t *testing.B) {
		create := make([]driver_mariadb.UserCreate, 10)
		for i := range len(create) {
			create[i] = driver_mariadb.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
		}

		users, err := dbMangoMariaDB.User.InsertMany(create)
		require.NoError(t, err)
		t.ResetTimer()

		for range t.N {
			for i := range len(create) {
				user, err := dbMangoMariaDB.User.FindById(users[i].Id)
				require.NoError(t, err)
				assert.Equal(t, users[i].Id, user.Id)
			}
		}
	})

	for _, value := range samples {
		t.Run("FindMany_"+strconv.Itoa(value), func(t *testing.B) {
			create := make([]driver_mariadb.UserCreate, value)
			ids := []uuid.UUID{}
			for i := range len(create) {
				create[i] = driver_mariadb.UserCreate{Name: fmt.Sprintf("John Doe %d", i), Email: fmt.Sprintf("john+%d@email.com", i)}
			}

			users, err := dbMangoMariaDB.User.InsertMany(create)
			for _, user := range users {
				ids = append(ids, user.Id)
			}
			require.NoError(t, err)
			t.ResetTimer()

			for range t.N {
				entries, err := dbMangoMariaDB.User.FindMany(
					dbMangoMariaDB.User.Query.Id.In(ids...),
				)
				require.NoError(t, err)
				assert.Len(t, entries, value)
			}
		})
	}
}
