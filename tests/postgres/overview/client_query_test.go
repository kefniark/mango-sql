package overview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAliasCount(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	b, err := db.User.Insert(UserCreate{Name: "bob"})
	assert.NoError(t, err)
	a, err := db.User.Insert(UserCreate{Name: "alice"})
	assert.NoError(t, err)

	_, err = db.Item.InsertMany([]ItemCreate{
		{Name: "item1", UserId: &a.Id, Quantity: 1},
		{Name: "item2", UserId: &a.Id, Quantity: 2},
		{Name: "item3", UserId: &a.Id, Quantity: 3},
		{Name: "item4", UserId: &b.Id, Quantity: 4},
		{Name: "item5", UserId: &b.Id, Quantity: 5},
	})
	assert.NoError(t, err)

	bData, err := db.Queries.UserItemsCount(db.User.Query.Id.Equal(b.Id))
	assert.NoError(t, err)

	aData, err := db.Queries.UserItemsCount(db.User.Query.Id.Equal(a.Id))
	assert.NoError(t, err)

	assert.Equal(t, "bob", bData[0].UsersName)
	assert.Equal(t, int64(2), bData[0].Count)
	assert.Equal(t, 9.0, bData[0].Sum)

	assert.Equal(t, "alice", aData[0].UsersName)
	assert.Equal(t, int64(3), aData[0].Count)
	assert.Equal(t, 6.0, aData[0].Sum)
}
