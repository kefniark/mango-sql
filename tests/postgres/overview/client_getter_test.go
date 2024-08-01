package overview

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFindMany(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.FindMany(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name ILIKE $1 OR name ILIKE $2", "%user1%", "%user2%")
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(users))
}

func TestFindManyWithFilters(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.FindMany(
		db.User.Query.Distinct(),

		// limit can be set before wheres
		db.User.Query.Limit(5),

		// can use as many query filters as needed
		db.User.Query.CreatedAt.Between(time.Now().Add(-360*24*time.Hour), time.Now()),
		db.User.Query.DeletedAt.IsNull(),

		// multiple orders : ORDER BY users.name DESC, users.id DESC
		db.User.Query.Name.OrderAsc(),
		db.User.Query.Id.OrderAsc(),
	)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(users))
}

func TestFind(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	users, err := db.User.FindMany(func(cond SelectBuilder) SelectBuilder {
		return cond.Offset(0).Limit(10)
	})

	assert.NoError(t, err)
	assert.Equal(t, 4, len(users))
}

func TestCount(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	count, err := db.User.Count()
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	count, err = db.User.Count(func(cond SelectBuilder) SelectBuilder {
		return cond.Where("name = ?", "user1")
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}
