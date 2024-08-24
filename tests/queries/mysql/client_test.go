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
)

//go:embed *.sql
var sqlPqFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	t.Helper()
	id := gonanoid.MustGenerate("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	testDB := "test_" + id

	dbAdmin, err := sqlx.Connect("mysql", "root:root@tcp(127.0.0.1:3307)/")
	dbAdmin.SetConnMaxIdleTime(time.Second * 10)
	assert.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDB))
	assert.NoError(t, err)

	_, err = dbAdmin.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO 'user'@'%%' WITH GRANT OPTION;", testDB))
	assert.NoError(t, err)

	_, err = dbAdmin.Exec("FLUSH PRIVILEGES;")
	assert.NoError(t, err)

	db, err := sqlx.Connect("mysql", fmt.Sprintf("user:password@tcp(127.0.0.1:3307)/%s?parseTime=true&multiStatements=true", testDB))
	db.SetConnMaxIdleTime(time.Second * 10)
	assert.NoError(t, err)

	fmt.Println("Create & Use DB", testDB)
	data, err := sqlPqFS.ReadFile("schema.sql")
	assert.NoError(t, err)

	_, err = db.Exec(string(data))
	assert.NoError(t, err)

	return New(db), func() {
		dbAdmin.MustExec(fmt.Sprintf("DROP DATABASE %s;", testDB))
		fmt.Println("Cleanup DB", testDB)
		dbAdmin.Close()
		db.Close()
	}
}

func TestInsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testInsert(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testInsert(t, tx)
		return errors.New("rollback")
	})
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testInsert(t *testing.T, db *DBClient) {
	user, err := db.User.Insert(UserCreate{
		Id:   1,
		Name: "tuna",
	})
	assert.NoError(t, err)
	assert.Equal(t, "tuna", user.Name)

	count, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	u, err := db.User.FindById(1)
	assert.NoError(t, err)
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
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))

	count, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 2, count)

	u, err := db.User.FindMany()
	assert.NoError(t, err)
	assert.Equal(t, "tuna", u[0].Name)
	assert.Equal(t, "salmon", u[1].Name)
}
*/

func TestUpdate(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testUpdate(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpdate(t, tx)
		return errors.New("rollback")
	})
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpdate(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	assert.NoError(t, err)

	u1, err := db.User.Update(UserUpdate{Id: 1, Name: "user1-updated"})
	assert.NoError(t, err)
	assert.Equal(t, "user1-updated", u1.Name)

	u2, err := db.User.FindById(1)
	assert.NoError(t, err)
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
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpdateMany(t *testing.T, db *DBClient) {
	ids, err := db.User.InsertMany([]UserCreate{
		{Id: 1, Name: "user1"},
		{Id: 2, Name: "user2"},
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(ids))

	_, err = db.User.UpdateMany([]UserUpdate{
		{Id: 1, Name: "user1-updated"},
		{Id: 2, Name: "user2-updated"},
	})
	assert.NoError(t, err)

	user, err := db.User.FindById(1)
	assert.NoError(t, err)

	assert.Equal(t, "user1-updated", user.Name)
}
*/

func TestUpsert(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testUpsert(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testUpsert(t, tx)
		return errors.New("rollback")
	})
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpsert(t *testing.T, db *DBClient) {
	_, err := db.User.Upsert(UserUpdate{Id: 1, Name: "usernew"})
	assert.NoError(t, err)

	_, err = db.User.Upsert(UserUpdate{Id: 1, Name: "user1-updated"})
	assert.NoError(t, err)

	count, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	user3, err := db.User.FindById(1)
	assert.NoError(t, err)
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
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testUpsertMany(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user1"})
	assert.NoError(t, err)

	_, err = db.User.UpsertMany([]UserUpdate{
		{Id: 1, Name: "usernew"},
		{Id: 2, Name: "user1-updated"},
	})
	assert.NoError(t, err)

	all, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 2, all)
}
*/

func TestSoftDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testSoftDelete(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testSoftDelete(t, tx)
		return errors.New("rollback")
	})
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testSoftDelete(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	assert.NoError(t, err)

	err = db.User.DeleteSoft(3)
	assert.NoError(t, err)
	count1, err := db.User.Count()
	assert.NoError(t, err)

	err = db.User.DeleteSoft(2)
	assert.NoError(t, err)
	count2, err := db.User.Count(
		db.User.Query.DeletedAt.IsNull(),
	)
	assert.NoError(t, err)

	assert.Equal(t, 1, count1)
	assert.Equal(t, 0, count2)
}

func TestHardDelete(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	testHardDelete(t, db)

	db2, close := newTestDB(t)
	defer close()
	err := db2.Transaction(func(tx *DBClient) error {
		testHardDelete(t, tx)
		return errors.New("rollback")
	})
	assert.Error(t, err)

	count, err := db2.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func testHardDelete(t *testing.T, db *DBClient) {
	_, err := db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	assert.NoError(t, err)

	err = db.User.DeleteHard(3)
	assert.NoError(t, err)
	count1, err := db.User.Count()
	assert.NoError(t, err)

	err = db.User.DeleteHard(2)
	assert.NoError(t, err)
	count2, err := db.User.Count()
	assert.NoError(t, err)

	assert.Equal(t, 1, count1)
	assert.Equal(t, 0, count2)
}

func TestTransaction(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	// transaction with rollback
	err := db.Transaction(func(tx *DBClient) error {
		_, err := tx.User.Insert(UserCreate{Id: 1, Name: "user1"})
		assert.NoError(t, err)

		return errors.New("rollback")
	})
	assert.Error(t, err)

	// transaction with commit
	err = db.Transaction(func(tx *DBClient) error {
		_, err := tx.User.Insert(UserCreate{Id: 2, Name: "user2"})
		assert.NoError(t, err)

		return nil
	})
	assert.NoError(t, err)

	all, err := db.User.FindMany()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(all))
	assert.Equal(t, "user2", all[0].Name)
}

func TestFind(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		assert.NoError(t, err)
	}

	// find by id
	user, err := db.User.FindById(1)
	assert.NoError(t, err)
	assert.Equal(t, "user1", user.Name)

	// find unique with filter
	user2, err := db.User.FindUnique(db.User.Query.Id.Equal(2))
	assert.NoError(t, err)
	assert.Equal(t, "user2", user2.Name)

	// find all
	users, err := db.User.FindMany()
	assert.NoError(t, err)
	assert.Equal(t, 5, len(users))

	// find with a filter
	filters, err := db.User.FindMany(
		db.User.Query.Id.GreaterThan(2),
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(filters))

	// limit / offset
	filters, err = db.User.FindMany(
		db.User.Query.Limit(2),
		db.User.Query.Offset(2),
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(filters))
	assert.Equal(t, "user3", filters[0].Name)
}

func TestFindLike(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		assert.NoError(t, err)
	}

	users, err := db.User.FindMany(db.User.Query.Name.Like("user%"))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(users))

	users2, err := db.User.FindMany(db.User.Query.Name.Like("%1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users2))

	count, err := db.User.Count(db.User.Query.Name.Like("%1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestFindIn(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		assert.NoError(t, err)
	}

	users, err := db.User.FindMany(db.User.Query.Name.In("user1", "user2"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))
}

func TestFindCustomFilter(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		assert.NoError(t, err)
	}

	// one custom filter
	users, err := db.User.FindMany(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name = ?", "user1")
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(users))

	// Mix multiple filters (custom and generated)
	users2, err := db.User.FindMany(
		func(cond SelectBuilder) SelectBuilder {
			return cond.Where("name LIKE ? OR name LIKE ?", "%user1%", "%user2%")
		},
		db.User.Query.Name.NotLike("%user3%"),
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users2))
}

func TestFindCustomQuery(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	for i := 1; i <= 5; i++ {
		_, err := db.User.Insert(UserCreate{Id: int64(i), Name: fmt.Sprintf("user%d", i)})
		assert.NoError(t, err)
	}

	err := db.User.DeleteSoft(3)
	assert.NoError(t, err)

	users, err := db.Queries.UserNotDeleted()
	assert.NoError(t, err)
	assert.Equal(t, 4, len(users))
}

func TestModel(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	// create a new user
	user := db.User.New()
	user.Id = 1
	user.Name = "bob"
	assert.NoError(t, user.Save(db))

	user2, err := db.User.FindById(1)
	assert.NoError(t, err)
	assert.Equal(t, "bob", user2.Name)

	// update the user
	user.Name = "alice"
	assert.NoError(t, user.Save(db))

	user3, err := db.User.FindById(1)
	assert.NoError(t, err)
	assert.Equal(t, "alice", user3.Name)
}

func TestFilters(t *testing.T) {
	db, close := newTestDB(t)
	defer close()
	_, err := db.User.Insert(UserCreate{Id: 1, Name: "user1"})
	assert.NoError(t, err)
	_, err = db.User.Insert(UserCreate{Id: 2, Name: "user2"})
	assert.NoError(t, err)

	u, err := db.User.Count(db.User.Query.Id.In(1))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.NotIn(1))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.Equal(1))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.NotEqual(1))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.IsNull())
	assert.NoError(t, err)
	assert.Equal(t, 0, u)

	u, err = db.User.Count(db.User.Query.Id.IsNotNull())
	assert.NoError(t, err)
	assert.Equal(t, 2, u)

	u2, err := db.User.FindMany(
		db.User.Query.Id.IsNotNull(),
		db.User.Query.Id.OrderAsc(),
		db.User.Query.Name.OrderDesc(),
	)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(u2))

	u, err = db.User.Count(db.User.Query.Id.GreaterThan(1))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.GreaterThanOrEqual(1))
	assert.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Id.LesserThan(2))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Id.LesserThanOrEqual(2))
	assert.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Id.Between(0, 3))
	assert.NoError(t, err)
	assert.Equal(t, 2, u)

	u, err = db.User.Count(db.User.Query.Name.Like("%1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)

	u, err = db.User.Count(db.User.Query.Name.NotLike("%1"))
	assert.NoError(t, err)
	assert.Equal(t, 1, u)
}
