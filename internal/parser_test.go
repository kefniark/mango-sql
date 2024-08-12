package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTableCreate(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);
	`)
	assert.NoError(t, err)

	// table
	assert.Equal(t, 1, len(schema.Tables))
	assert.Equal(t, "orders", schema.Tables["orders"].Name)

	// columns
	assert.Equal(t, 2, len(schema.Tables["orders"].Columns))
	assert.Equal(t, "id", schema.Tables["orders"].Columns["id"].Name)
	assert.Equal(t, "uuid", schema.Tables["orders"].Columns["id"].Type)
	assert.Equal(t, "name", schema.Tables["orders"].Columns["name"].Name)
	assert.Equal(t, "string", schema.Tables["orders"].Columns["name"].Type)

	// primary index
	assert.Equal(t, 1, len(schema.Tables["orders"].Constraints))
	assert.Equal(t, 1, len(schema.Tables["orders"].Indexes))
}

func TestParseUnique(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL UNIQUE
	);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables))

	assert.Equal(t, "UNIQUE", schema.Tables["orders"].Constraints[1].Type)
	assert.Equal(t, "name", schema.Tables["orders"].Constraints[1].Columns[0])
}

func TestParseUniqueMulti(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL,
		UNIQUE(id, name)
	);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables))

	assert.Equal(t, "UNIQUE", schema.Tables["orders"].Constraints[1].Type)
	assert.Equal(t, "id", schema.Tables["orders"].Constraints[1].Columns[0])
	assert.Equal(t, "name", schema.Tables["orders"].Constraints[1].Columns[1])
}

func TestParseUniqueAlter(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders ADD CONSTRAINT fk_order_id UNIQUE (id, name);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables))

	assert.Equal(t, "UNIQUE", schema.Tables["orders"].Constraints[1].Type)
	assert.Equal(t, "id", schema.Tables["orders"].Constraints[1].Columns[0])
	assert.Equal(t, "name", schema.Tables["orders"].Constraints[1].Columns[1])
}

func TestParseUniqueDrop(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders ADD CONSTRAINT fk_order_id UNIQUE (id, name);
	ALTER TABLE orders DROP CONSTRAINT fk_order_id;
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables))
	assert.Equal(t, 1, len(schema.Tables["orders"].Constraints))
}

func TestParseRef(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	CREATE TABLE orders_items (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL,
		order_id	UUID REFERENCES orders(id)
	);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(schema.Tables))
	assert.Equal(t, 1, len(schema.Tables["orders_items"].References))

	assert.Equal(t, "order_id", schema.Tables["orders_items"].References[0].Columns[0])
	assert.Equal(t, "orders", schema.Tables["orders_items"].References[0].Table)
	assert.Equal(t, "id", schema.Tables["orders_items"].References[0].TableColumns[0])
}

func TestParseAlterRef(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	CREATE TABLE orders_items (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL,
		order_id	UUID
	);

	ALTER TABLE orders_items ADD CONSTRAINT fk_order_id FOREIGN KEY (order_id) REFERENCES orders(id);
	`)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(schema.Tables))
	assert.Equal(t, 1, len(schema.Tables["orders_items"].References))
	assert.Equal(t, "order_id", schema.Tables["orders_items"].References[0].Columns[0])
	assert.Equal(t, "orders", schema.Tables["orders_items"].References[0].Table)
	assert.Equal(t, "id", schema.Tables["orders_items"].References[0].TableColumns[0])
}

func TestParseTableRename(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders RENAME TO new_orders;
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables))
	assert.NotNil(t, schema.Tables["new_orders"].Columns["name"])
}

func TestParseTableDrop(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	DROP TABLE orders;
	`)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(schema.Tables))
}

func TestParseAlterTableColumnAdd(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders ADD COLUMN created_at TIMESTAMP;
	`)
	assert.NoError(t, err)

	// column
	assert.NotNil(t, schema.Tables["orders"].Columns["created_at"])

	assert.Equal(t, "created_at", schema.Tables["orders"].Columns["created_at"].Name)
	assert.Equal(t, "timestamp", schema.Tables["orders"].Columns["created_at"].Type)
}

func TestParseAlterTableColumnType(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders ALTER COLUMN name TYPE VARCHAR(255);
	`)
	assert.NoError(t, err)

	// column
	assert.Equal(t, "name", schema.Tables["orders"].Columns["name"].Name)
	assert.Equal(t, "varchar", schema.Tables["orders"].Columns["name"].Type)
}

func TestParseAlterTableColumnRename(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders RENAME COLUMN name TO new_name;
	`)
	assert.NoError(t, err)

	// column
	assert.Nil(t, schema.Tables["orders"].Columns["name"])
	assert.Equal(t, "name", schema.Tables["orders"].Columns["new_name"].Name)
	assert.Equal(t, "string", schema.Tables["orders"].Columns["new_name"].Type)
}

func TestParseAlterColumnDrop(t *testing.T) {
	schema, err := ParseSchema(`
	CREATE TABLE orders (
		id          UUID  PRIMARY KEY,
		name        text  NOT NULL
	);

	ALTER TABLE orders DROP COLUMN name;
	`)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(schema.Tables["orders"].Columns))
	assert.Nil(t, schema.Tables["orders"].Columns["name"])
}
