package postgres

/**
	Auto-Generated by Mango SQL
	DO NOT EDIT!
	Url: https://github.com/kefniark/mangosql
	Date: 2024-07-19 09:24:23.463517404 +0000 UTC m=+0.018570489
**/

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"database/sql"
	squirrel "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/google/uuid"
)

type SelectBuilder = squirrel.SelectBuilder
type DBClient struct {
	ctx  *DBContext
	User *UserQueries
}

func newClient(ctx *DBContext) *DBClient {
	return &DBClient{
		ctx:  ctx,
		User: &UserQueries{ctx: ctx},
	}
}

func New(db *sqlx.DB) *DBClient {
	client := newClient(&DBContext{
		db:       db,
		prepared: make(map[string]*sqlx.Stmt),
		tx:       nil,
	})
	return client
}

func (db DBClient) Transaction(transaction func(dbClient *DBClient) error) (e error) {
	if db.ctx.tx != nil {
		return fmt.Errorf("nested transaction is not supported")
	}

	tx, err := db.ctx.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			e = errors.New(p.(string))
		}
	}()

	client := newClient(&DBContext{
		db:       db.ctx.db,
		prepared: db.ctx.prepared,
		tx:       tx,
	})
	if err = transaction(client); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// Split slice into chunks of the given size
func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

func (ctx *DBContext) Get(dest interface{}, query string, args ...interface{}) error {
	if ctx.tx != nil {
		return ctx.tx.Get(dest, query, args...)
	}
	return ctx.db.Get(dest, query, args...)
}

func (ctx *DBContext) Select(dest interface{}, query string, args ...interface{}) error {
	if ctx.tx != nil {
		return ctx.tx.Select(dest, query, args...)
	}
	return ctx.db.Select(dest, query, args...)
}

func (ctx *DBContext) Exec(query string, args ...any) (sql.Result, error) {
	if ctx.tx != nil {
		return ctx.tx.Exec(query, args...)
	}
	return ctx.db.Exec(query, args...)
}

func (ctx *DBContext) Stmt(query string) (*sqlx.Stmt, error) {
	if _, ok := ctx.prepared[query]; !ok {
		stmt, err := ctx.db.Preparex(query)
		if err != nil {
			return nil, err
		}
		ctx.prepared[query] = stmt
	}

	stmt := ctx.prepared[query]
	if ctx.tx != nil {
		return ctx.tx.Stmtx(stmt), nil
	}
	return stmt, nil
}

const (
	// users table
	userTable         = `users`
	userCreateSql     = `INSERT INTO users (id, email, name) VALUES ($1, $2, $3) RETURNING id, email, name, created_at, updated_at, deleted_at`
	userCreateManySql = `INSERT INTO users (id, email, name) SELECT * FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) RETURNING users.id`
	userUpsertSql     = `INSERT INTO users (id, email, name) VALUES ($1, $2, $3) ON CONFLICT(id) DO UPDATE SET email=EXCLUDED.email, name=EXCLUDED.name RETURNING id, email, name, created_at, updated_at, deleted_at`
	userUpsertManySql = `INSERT INTO users (id, email, name) SELECT * FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) ON CONFLICT(id) DO UPDATE SET email=EXCLUDED.email, name=EXCLUDED.name RETURNING users.id`
	userUpdateSql     = `UPDATE users SET email=$2, name=$3 WHERE id=$1 RETURNING id, email, name, created_at, updated_at, deleted_at`
	userUpdateManySql = `UPDATE users SET email=t.email, name=t.name FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) WHERE users.id=t.id RETURNING users.id`
	userDeleteSoftSql = `UPDATE users SET deleted_at=NOW() WHERE id=$1`
	userDeleteHardSql = `DELETE FROM users WHERE id=$1`
	userGetByIdSql    = `SELECT id, email, name, created_at, updated_at, deleted_at FROM users WHERE id=$1 LIMIT 1`
)

type DBContext struct {
	db       *sqlx.DB
	tx       *sqlx.Tx
	prepared map[string]*sqlx.Stmt
}

// Table users

var UserFields = []string{"id", "email", "name", "created_at", "updated_at", "deleted_at"}

type UserPrimaryKey = uuid.UUID

type UserModel struct {
	Id        uuid.UUID    `json:"id" db:"id"`
	Email     string       `json:"email" db:"email"`
	Name      string       `json:"name" db:"name"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at" db:"deleted_at"`
}

type UserCreate struct {
	Email string `json:"email" db:"email"`
	Name  string `json:"name" db:"name"`
}

type UserUpdate struct {
	Id    uuid.UUID `json:"id" db:"id"`
	Email string    `json:"email" db:"email"`
	Name  string    `json:"name" db:"name"`
}

// Helper used to manipulate User (users table)
type UserQueries struct {
	ctx *DBContext
}

// Insert a User and return the created row
//
// sql: INSERT INTO users (id, email, name) VALUES ($1, $2, $3) RETURNING id, email, name, created_at, updated_at, deleted_at
func (q *UserQueries) Create(input UserCreate) (*UserModel, error) {
	stmt, err := q.ctx.Stmt(userCreateSql)
	if err != nil {
		return nil, err
	}

	data := &UserModel{}
	if err := stmt.Get(data, uuid.New(), input.Email, input.Name); err != nil {
		return nil, err
	}
	return data, nil
}

// Batch Insert User
//
// sql: INSERT INTO users (id, email, name) SELECT * FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) RETURNING users.id
func (q *UserQueries) CreateMany(inputs []UserCreate) ([]UserPrimaryKey, error) {
	stmt, err := q.ctx.Stmt(userCreateManySql)
	if err != nil {
		return nil, err
	}

	ids := []UserPrimaryKey{}
	for _, chunk := range chunkBy(inputs, 250) {
		records := make([]UserUpdate, len(chunk))
		for i, input := range chunk {
			records[i] = UserUpdate{
				Id:    uuid.New(),
				Email: input.Email,
				Name:  input.Name,
			}
		}

		data, err := json.Marshal(records)
		if err != nil {
			return nil, err
		}

		loopIds := []UserPrimaryKey{}
		if err = stmt.Select(&loopIds, data); err != nil {
			return nil, err
		}
		ids = append(ids, loopIds...)
	}

	return ids, nil
}

// Upsert a User (create or update if already exist) and return the updated row
//
// sql: INSERT INTO users (id, email, name) VALUES ($1, $2, $3) ON CONFLICT(id) DO UPDATE SET email=EXCLUDED.email, name=EXCLUDED.name RETURNING id, email, name, created_at, updated_at, deleted_at
func (q *UserQueries) Upsert(input UserUpdate) (*UserModel, error) {
	stmt, err := q.ctx.Stmt(userUpsertSql)
	if err != nil {
		return nil, err
	}

	data := &UserModel{}
	if err := stmt.Get(data, input.Id, input.Email, input.Name); err != nil {
		return nil, err
	}
	return data, nil
}

// Batch Upsert User (create or update if already exist)
//
// sql: INSERT INTO users (id, email, name) SELECT * FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) ON CONFLICT(id) DO UPDATE SET email=EXCLUDED.email, name=EXCLUDED.name RETURNING users.id
func (q *UserQueries) UpsertMany(inputs []UserUpdate) ([]UserPrimaryKey, error) {
	stmt, err := q.ctx.Stmt(userUpsertManySql)
	if err != nil {
		return nil, err
	}

	ids := []UserPrimaryKey{}
	for _, chunk := range chunkBy(inputs, 250) {
		data, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}

		loopIds := []UserPrimaryKey{}
		if err = stmt.Select(&loopIds, data); err != nil {
			return nil, err
		}
		ids = append(ids, loopIds...)
	}

	return ids, nil
}

// Update a User and return the updated row
//
// sql: UPDATE users SET email=$2, name=$3 WHERE id=$1 RETURNING id, email, name, created_at, updated_at, deleted_at
func (q *UserQueries) Update(input UserUpdate) (*UserModel, error) {
	stmt, err := q.ctx.Stmt(userUpdateSql)
	if err != nil {
		return nil, err
	}

	data := &UserModel{}
	if err := stmt.Get(data, input.Id, input.Email, input.Name); err != nil {
		return nil, err
	}
	return data, nil
}

// Batch Update User
//
// sql: UPDATE users SET email=t.email, name=t.name FROM jsonb_to_recordset($1) as t(id UUID, email VARCHAR(64), name VARCHAR(64)) WHERE users.id=t.id RETURNING users.id
func (q *UserQueries) UpdateMany(inputs []UserUpdate) ([]UserPrimaryKey, error) {
	stmt, err := q.ctx.Stmt(userUpdateManySql)
	if err != nil {
		return nil, err
	}

	ids := []UserPrimaryKey{}
	for _, chunk := range chunkBy(inputs, 250) {
		data, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}

		loopIds := []UserPrimaryKey{}
		if err = stmt.Select(&loopIds, data); err != nil {
			return nil, err
		}
		ids = append(ids, loopIds...)
	}

	return ids, nil
}

// Delete a User (soft delete, data are still in the database)
//
// sql: UPDATE users SET deleted_at=NOW() WHERE id=$1
func (q *UserQueries) DeleteSoft(id uuid.UUID) error {
	stmt, err := q.ctx.Stmt(userDeleteSoftSql)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id)
	return err
}

// Delete a User (hard delete, data are removed from the database)
//
// sql: DELETE FROM users WHERE id=$1
func (q *UserQueries) DeleteHard(id uuid.UUID) error {
	stmt, err := q.ctx.Stmt(userDeleteHardSql)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id)
	return err
}

// Count all User records
func (q *UserQueries) Count() (int, error) {
	return q.CountWhere(func(cond SelectBuilder) SelectBuilder {
		return cond
	})
}

// Count User records based on filter conditions
func (q *UserQueries) CountWhere(where func(cond SelectBuilder) SelectBuilder) (int, error) {
	query := squirrel.Select("count(id)").From(userTable).PlaceholderFormat(squirrel.Dollar)
	sql, args, err := where(query).ToSql()
	if err != nil {
		return 0, err
	}

	count := 0
	err = q.ctx.Get(&count, sql, args...)
	return count, err
}

// Find User records based on the provided conditions
func (q *UserQueries) Where(where func(cond SelectBuilder) SelectBuilder) ([]UserModel, error) {
	query := squirrel.Select(UserFields...).From(userTable).PlaceholderFormat(squirrel.Dollar)
	sql, args, err := where(query).ToSql()
	if err != nil {
		return nil, err
	}

	entries := []UserModel{}
	err = q.ctx.Select(&entries, sql, args...)
	return entries, err
}

// List all User records
func (q *UserQueries) All(offset, limit uint64) ([]UserModel, error) {
	sql, _, err := squirrel.Select(UserFields...).OrderBy("id ASC").From(userTable).PlaceholderFormat(squirrel.Dollar).Offset(offset).Limit(limit).ToSql()
	if err != nil {
		return nil, err
	}

	entries := []UserModel{}
	err = q.ctx.Select(&entries, sql)
	return entries, err
}

// Find User
//
// sql: SELECT id, email, name, created_at, updated_at, deleted_at FROM users WHERE id=$1 LIMIT 1
func (q *UserQueries) GetById(id uuid.UUID) (*UserModel, error) {
	stmt, err := q.ctx.Stmt(userGetByIdSql)
	if err != nil {
		return nil, err
	}

	data := &UserModel{}
	if err := stmt.Get(data, id); err != nil {
		return nil, err
	}
	return data, nil
}
