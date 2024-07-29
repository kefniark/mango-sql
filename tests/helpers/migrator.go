package helpers

import (
	"context"
	"database/sql"

	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/common"
)

type SchemaMigrator struct {
	Data []byte
}

func (s *SchemaMigrator) Hash() (string, error) {
	hash := common.NewRecursiveHash()
	hash.Add(s.Data)
	return hash.String(), nil
}

func (s *SchemaMigrator) Prepare(context.Context, *sql.DB, pgtestdb.Config) error {
	return nil
}

func (s *SchemaMigrator) Migrate(ctx context.Context, db *sql.DB, config pgtestdb.Config) error {
	_, err := db.Exec(string(s.Data))
	return err
}

func (s *SchemaMigrator) Verify(context.Context, *sql.DB, pgtestdb.Config) error {
	return nil
}

type Migrator interface {
	// Hash should return a unique identifier derived from the state of the database
	// after it has been fully migrated. For instance, it may return a hash of all
	// of the migration names and contents.
	//
	// pgtestdb will use the returned Hash to identify a template database. If a
	// Migrator returns a Hash that has already been used to create a template
	// database, it is assumed that the database need not be recreated since it
	// would result in the same schema and data.
	Hash() (string, error)

	// Prepare should perform any plugin or extension installations necessary to
	// make the database ready for the migrations. For instance, you may want to
	// enable certain extensions like `trigram` or `pgcrypto`, or creating or
	// altering certain roles and permissions.
	// Prepare will be given a *sql.DB connected to the template database.
	Prepare(context.Context, *sql.DB, pgtestdb.Config) error

	// Migrate is a function that actually performs the schema and data
	// migrations to provision a template database. The connection given to this
	// function is to an entirely new, empty, database. Migrate will be called
	// only once, when the template database is being created.
	Migrate(context.Context, *sql.DB, pgtestdb.Config) error

	// Verify is called each time you ask for a new test database instance. It
	// should be cheaper than the call to Migrate(), and should return nil iff
	// the database is in the correct state. An example implementation would be
	// to check that all the migrations have been marked as applied, and
	// otherwise return an error.
	Verify(context.Context, *sql.DB, pgtestdb.Config) error
}
