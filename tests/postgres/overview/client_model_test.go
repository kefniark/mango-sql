package overview

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelCreate(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user := db.User.New()
	user.Name = "bob"
	user.Email = "bob@email.com"
	assert.NoError(t, user.Save(db))

	user2, err := db.User.FindById(user.Id)
	assert.NoError(t, err)

	assert.Equal(t, user.Email, user2.Email)
}

func TestModelSave(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	user, err := db.User.Insert(UserCreate{Name: "user1", Email: "user1@email.com"})
	assert.NoError(t, err)

	originalName := user.Name
	user.Name = "user1-up"
	err = user.Save(db)

	assert.NoError(t, err)

	user2, err := db.User.FindById(user.Id)
	assert.NoError(t, err)

	assert.Equal(t, "user1-up", user2.Name)
	assert.NotEqual(t, originalName, user.Name)
}
