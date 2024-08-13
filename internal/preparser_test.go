package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrigger(t *testing.T) {
	schema, err := ParseSchema(`
		CREATE TABLE actor (
  			actor_id numeric NOT NULL ,
			first_name VARCHAR(45) NOT NULL,
			last_name VARCHAR(45) NOT NULL,
			last_update TIMESTAMP NOT NULL,
			PRIMARY KEY  (actor_id)
  		);

		CREATE  INDEX idx_actor_last_name ON actor(last_name);
 
		CREATE TRIGGER actor_trigger_ai AFTER INSERT ON actor
		BEGIN
			UPDATE actor SET last_update = DATETIME('NOW')  WHERE rowid = new.rowid;
		END;
	`)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(schema.Tables["actor"].Columns))
}

func TestEnums(t *testing.T) {
	schema, err := ParseSchema(`
		CREATE TABLE actor (
  			actor_id numeric NOT NULL ,
			first_name VARCHAR(45) NOT NULL,
			last_name VARCHAR(45) NOT NULL,
			last_update TIMESTAMP NOT NULL,
			PRIMARY KEY  (actor_id)
  		);

		CREATE  INDEX idx_actor_last_name ON actor(last_name);
 
		CREATE TYPE mood AS ENUM ('sad', 'ok', 'happy');
	`)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(schema.Tables["actor"].Columns))
}

func TestBlob(t *testing.T) {
	schema, err := ParseSchema(`
		CREATE TABLE actor (
  			actor_id numeric NOT NULL ,
			description BLOB SUB_TYPE TEXT DEFAULT NULL,
			first_name VARCHAR(45) NOT NULL,
			last_name VARCHAR(45) NOT NULL,
			last_update TIMESTAMP NOT NULL,
			PRIMARY KEY  (actor_id)
  		);

		CREATE  INDEX idx_actor_last_name ON actor(last_name);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 5, len(schema.Tables["actor"].Columns))
}
