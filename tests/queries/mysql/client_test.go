package mysql

import (
	"embed"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../../cmd/mangosql/ --output ./client.go --package mysql --driver mysql --logger console ./schema.sql

//go:embed *.sql
var sqlPqFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	t.Helper()
	id := gonanoid.MustGenerate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	testDB := "test_" + id

	dbAdmin, err := sqlx.Connect("mysql", "root:root@tcp(127.0.0.1:3307)/")
	dbAdmin.SetConnMaxIdleTime(time.Second * 10)
	require.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDB))
	require.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO 'user'@'%%' WITH GRANT OPTION;", testDB))
	require.NoError(t, err)

	_, err = dbAdmin.Exec("FLUSH PRIVILEGES;")
	require.NoError(t, err)

	db, err := sqlx.Connect("mysql", fmt.Sprintf("user:password@tcp(127.0.0.1:3307)/%s?parseTime=true&multiStatements=true", testDB))
	db.SetConnMaxIdleTime(time.Second * 10)
	require.NoError(t, err)

	fmt.Println("Create & Use DB", testDB)
	data, err := sqlPqFS.ReadFile("schema.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(data))
	require.NoError(t, err)

	return New(db), func() {
		dbAdmin.MustExec(fmt.Sprintf("DROP DATABASE %s;", testDB))
		fmt.Println("Cleanup DB", testDB)
		dbAdmin.Close()
		db.Close()
	}
}

func TestInsert(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	testInsert(t, db)

	db2, closeDB := newTestDB(t)
	defer closeDB()
	err := db2.Transaction(func(tx *DBClient) error {
		testInsert(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testInsert(t *testing.T, db *DBClient) {
	user, err := db.User.Insert(UserCreate{
		Id:   1,
		Name: "tuna",
	})
	require.NoError(t, err)
	assert.Equal(t, "tuna", user.Name)

	count, err := db.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	u, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "tuna", u.Name)
}

/*
func TestInsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testInsertMany(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testInsertMany(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testInsertMany(t *testing.T, db *DBClient) {
	users, err := db.User.InsertMany([]UserCreate{
		{
			Id:   1,
			Name: "tuna",
		},
		{
			Id:   2,
			Name: "salmon",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(users))

	count, err := db.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	u, err := db.User.FindMany()
	require.NoError(t, err)
	assert.Equal(t, "tuna", u[0].Name)
	assert.Equal(t, "salmon", u[1].Name)
}
*/

func TestUpdate(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	testUpdate(t, db)

	db2, closeDB := newTestDB(t)
	defer closeDB()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpdate(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpdate(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	require.NoError(t, err)

	u1, err := db.User.Update(UserUpdate{Id: 1, Name: "user1-updated"})
	require.NoError(t, err)
	assert.Equal(t, "user1-updated", u1.Name)

	u2, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "user1-updated", u2.Name)
}

/*
func TestUpdateMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testUpdateMany(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpdateMany(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpdateMany(t *testing.T, db *DBClient) {
	ids, err := db.User.InsertMany([]UserCreate{
		{Id: 1, Name: "user1"},
		{Id: 2, Name: "user2"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, len(ids))

	_, err = db.User.UpdateMany([]UserUpdate{
		{Id: 1, Name: "user1-updated"},
		{Id: 2, Name: "user2-updated"},
	})
	require.NoError(t, err)

	user, err := db.User.FindById(1)
	require.NoError(t, err)

	assert.Equal(t, "user1-updated", user.Name)
}
*/

func TestUpsert(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	testUpsert(t, db)

	db2, closeDB := newTestDB(t)
	defer closeDB()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpsert(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpsert(t *testing.T, db *DBClient) {
	_, err := db.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
	require.NoError(t, err)

	_, err = db.User.Upsert(UserUpdate{Id: 1, Name: "user1-updated"})
	require.NoError(t, err)

	count, err := db.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	user3, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "user1-updated", user3.Name)
}

/*
func TestUpsertMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testUpsertMany(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpsertMany(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpsertMany(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user1"})
	require.NoError(t, err)

	_, err = db.User.UpsertMany([]UserUpdate{
		{Id: 1, Name: "usernew"},
		{Id: 2, Name: "user1-updated"},
	})
	require.NoError(t, err)

	all, err := db.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 2, all)
}
*/

func TestSoftDelete(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	testSoftDelete(t, db)

	db2, closeDB := newTestDB(t)
	defer closeDB()
	err := db2.Transaction(func(tx *DBClient) error {
		testSoftDelete(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testSoftDelete(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	require.NoError(t, err)

	err = db.User.DeleteSoft(3)
	require.NoError(t, err)
	count1, err := db.User.Count()
	require.NoError(t, err)

	err = db.User.DeleteSoft(2)
	require.NoError(t, err)
	count2, err := db.User.Count(
		db.User.Query.DeletedAt.IsNull(),
	)
	require.NoError(t, err)

	assert.Equal(t, 1, count1)
	assert.Equal(t, 0, count2)
}

func TestHardDelete(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	testHardDelete(t, db)

	db2, closeDB := newTestDB(t)
	defer closeDB()
	err := db2.Transaction(func(tx *DBClient) error {
		testHardDelete(t, tx)
		return errors.New("rollback")
	})
	require.Error(t, err)

	count, err := db2.User.Count()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testHardDelete(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	require.NoError(t, err)

	err = db.User.DeleteHard(3)
	require.NoError(t, err)
	count1, err := db.User.Count()
	require.NoError(t, err)

	err = db.User.DeleteHard(2)
	require.NoError(t, err)
	count2, err := db.User.Count()
	require.NoError(t, err)

	assert.Equal(t, 1, count1)
	assert.Equal(t, 0, count2)
}

func TestTransaction(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	// transaction with rollback
	err := db.Transaction(func(tx *DBClient) error {
		_, err := tx.User.Insert(UserCreate{Id: 1, Name: "user1"})
		require.NoError(t, err)

		return errors.New("rollback")
	})
	require.Error(t, err)

	// transaction with commit
	err = db.Transaction(func(tx *DBClient) error {
		_, err := tx.User.Insert(UserCreate{Id: 2, Name: "user2"})
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	all, err := db.User.FindMany()
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, "user2", all[0].Name)
}

func TestFind(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		require.NoError(t, err)
	}

	// find by id
	user, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "user1", user.Name)

	// find unique with filter
	user2, err := db.User.FindUnique(db.User.Query.Id.Equal(2))
	require.NoError(t, err)
	assert.Equal(t, "user2", user2.Name)

	// find all
	users, err := db.User.FindMany()
	require.NoError(t, err)
	assert.Len(t, users, 5)

	// find with a filter
	filters, err := db.User.FindMany(
		db.User.Query.Id.GreaterThan(2),
	)
	require.NoError(t, err)
	assert.Len(t, filters, 3)

	// limit / offset
	filters, err = db.User.FindMany(
		db.User.Query.Limit(2),
		db.User.Query.Offset(2),
	)
	require.NoError(t, err)
	assert.Len(t, filters, 2)
	assert.Equal(t, "user3", filters[0].Name)
}

func TestFindLike(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		require.NoError(t, err)
	}

	users, err := db.User.FindMany(db.User.Query.Name.Like("user%"))
	require.NoError(t, err)
	assert.Len(t, users, 5)

	users2, err := db.User.FindMany(db.User.Query.Name.Like("%1"))
	require.NoError(t, err)
	assert.Len(t, users2, 1)

	count, err := db.User.Count(db.User.Query.Name.Like("%1"))
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFindIn(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		require.NoError(t, err)
	}

	users, err := db.User.FindMany(db.User.Query.Name.In("user1", "user2"))
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestFindCustomFilter(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		require.NoError(t, err)
	}

	// one custom filter
	users, err := db.User.FindMany(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name = ?", "user1")
	})
	require.NoError(t, err)
	assert.Len(t, users, 1)

	// Mix multiple filters (custom and generated)
	users2, err := db.User.FindMany(
		func(cond SelectBuilder) SelectBuilder {
			return cond.Where("name LIKE ? OR name LIKE ?", "%user1%", "%user2%")
		},
		db.User.Query.Name.NotLike("%user3%"),
	)
	require.NoError(t, err)
	assert.Len(t, users2, 2)
}

func TestFindCustomQuery(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		require.NoError(t, err)
	}

	err := db.User.DeleteSoft(3)
	require.NoError(t, err)

	users, err := db.Queries.UserNotDeleted()
	require.NoError(t, err)
	assert.Len(t, users, 4)
}

func TestModel(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()

	// create a new user
	user := db.User.New()
	user.Id = 1
	user.Name = "bob"
	require.NoError(t, user.Save(db))

	user2, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "bob", user2.Name)

	// update the user
	user.Name = "alice"
	require.NoError(t, user.Save(db))

	user3, err := db.User.FindById(1)
	require.NoError(t, err)
	assert.Equal(t, "alice", user3.Name)
}

func TestFilters(t *testing.T) {
	db, closeDB := newTestDB(t)
	defer closeDB()
	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	require.NoError(t, err)
	_, err = db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	require.NoError(t, err)

	u, err := db.User.Count(db.User.Query.Id.In(1))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.NotIn(1))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.Equal(1))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.NotEqual(1))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.IsNull())
	require.NoError(t, err)
	assert.Equal(t, 0, u)

	u, err = db.User.Count(db.User.Query.Id.IsNotNull())
	require.NoError(t, err)
	assert.Equal(t, 2, u)

	u2, err := db.User.FindMany(
		db.User.Query.Id.IsNotNull(),
		db.User.Query.Id.OrderAsc(),
		db.User.Query.Name.OrderDesc(),
	)
	require.NoError(t, err)
	assert.Len(t, u2, 2)

	u, err = db.User.Count(db.User.Query.Id.GreaterThan(1))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.GreaterThanOrEqual(1))
	require.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Id.LesserThan(2))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.LesserThanOrEqual(2))
	require.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Id.Between(0, 3))
	require.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Name.Like("%1"))
	require.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Name.NotLike("%1"))
	require.NoError(t, err)
	assert.Equal(t, 1, u)
}
