package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	assert.Len(t, schema.Tables["actor"].Columns, 4)
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
	require.NoError(t, err)

	assert.Len(t, schema.Tables["actor"].Columns, 4)
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
	require.NoError(t, err)

	assert.Len(t, schema.Tables["actor"].Columns, 5)
}
