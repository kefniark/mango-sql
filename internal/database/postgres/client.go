package database

/**
 * This file was auto-generated by Mango SQL : https://github.com/kefniark/mangosql
 * Do not make direct changes to the file.
 */

import (
	"context"
	"errors"
	"fmt"

	"log"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
)

type SelectBuilder = squirrel.SelectBuilder
type WhereCondition = func(query SelectBuilder) SelectBuilder

var placeholder = squirrel.Dollar

type DBClient struct {
	ctx *DBContext
	// User Custom SQL Queries
	//
	// Usage:
	//   entities, err := db.Queries.MySqlRequests()
	Queries *CustomQueries
}

// Create a new instance of MangoSql
func New(db DBPgx) *DBClient {
	return newClient(&DBContext{
		db: db,
		tx: nil,
	})
}

func newClient(ctx *DBContext) *DBClient {
	return &DBClient{
		ctx: ctx,
		// Models
		// Custom Queries
		Queries: &CustomQueries{ctx: ctx}}
}

type FilterGenericField[T any] struct {
	table string
	field string
}

// Only include Records with a field contains in a set of values
func (f FilterGenericField[T]) In(args ...T) WhereCondition {
	sql := fmt.Sprintf("%s.%s = ANY(?)", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql, pq.Array(args))
	}
}

// Exclude Records with a field not contains in a set of values
func (f FilterGenericField[T]) NotIn(args ...T) WhereCondition {
	sql := fmt.Sprintf("%s.%s != ANY(?)", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql, pq.Array(args))
	}
}

// Only include Records with a field specific value
func (f FilterGenericField[T]) Equal(arg T) WhereCondition {
	sql := fmt.Sprintf("%s.%s = ?", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql, arg)
	}
}

// Exclude Records with a field specific value
func (f FilterGenericField[T]) NotEqual(arg T) WhereCondition {
	sql := fmt.Sprintf("%s.%s != ?", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql, arg)
	}
}

// Only include Records with a field has undefined value
func (f FilterGenericField[T]) IsNull() WhereCondition {
	sql := fmt.Sprintf("%s.%s IS NULL", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql)
	}
}

// Only include Records with a field has defined values
func (f FilterGenericField[T]) IsNotNull() WhereCondition {
	sql := fmt.Sprintf("%s.%s IS NOT NULL", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.Where(sql)
	}
}

// Sort Records in ASC order
func (f FilterGenericField[T]) OrderAsc() WhereCondition {
	sql := fmt.Sprintf("%s.%s ASC", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.OrderBy(sql)
	}
}

// Sort Records in DESC order
func (f FilterGenericField[T]) OrderDesc() WhereCondition {
	sql := fmt.Sprintf("%s.%s DESC", f.table, f.field)
	return func(cond SelectBuilder) SelectBuilder {
		return cond.OrderBy(sql)
	}
}

type DBPgx interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Create a new Sql transaction.
// If any error or panic occurs inside, the transaction is automatically rollback
//
// Usage:
//
//	err := db.Transaction(func(tx *DBClient) error {
//	  // ... can use tx. like db.
//	})
func (db DBClient) Transaction(transaction func(dbClient *DBClient) error) (e error) {
	if db.ctx.tx != nil {
		return errors.New("nested transaction is not supported")
	}

	tx, err := db.ctx.db.Begin(context.Background())
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			e = errors.Join(errors.New(p.(string)), tx.Rollback(context.Background()))
		}
	}()

	client := newClient(&DBContext{
		db: db.ctx.db,
		tx: tx,
	})
	if err = transaction(client); err != nil {
		return errors.Join(err, tx.Rollback(context.Background()))
	}

	return tx.Commit(context.Background())
}

// Split slice into chunks of the given size
func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

// Execute a Custom SQL query and get one row result.
//
// Usage:
//
//	data MyResult
//	res, err := db.QueryRow(ctx, &data, sql, args...)
func QueryRow(ctx *DBContext, data interface{}, sql string, args ...interface{}) error {
	db := ctx.db
	if ctx.tx != nil {
		db = ctx.tx
	}

	return db.QueryRow(context.Background(), sql, args...).Scan(data)
}

// Execute a Custom SQL query and get one row result.
//
// Usage:
//
//	res, err := db.QueryOne[MyResult](ctx, sql, args...)
func QueryOne[T any](ctx *DBContext, sql string, args ...interface{}) (*T, error) {
	db := ctx.db
	if ctx.tx != nil {
		db = ctx.tx
	}

	rows, err := db.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[T])
}

// Execute a Custom SQL query and get many rows result.
//
// Usage:
//
//	res, err := db.QueryMany[MyResult](ctx, sql, args...)
func QueryMany[T any](ctx *DBContext, sql string, args ...interface{}) ([]T, error) {
	db := ctx.db
	if ctx.tx != nil {
		db = ctx.tx
	}

	rows, err := db.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}

// Execute a Custom SQL query without result.
//
// Usage:
//
//	err := db.Exec(ctx, sql, args...)
func Exec(ctx *DBContext, sql string, args ...interface{}) error {
	db := ctx.db
	if ctx.tx != nil {
		db = ctx.tx
	}

	_, err := db.Exec(context.Background(), sql, args...)
	return err
}

func first[T any](items []T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, errors.New("first element not found")
	}

	return &items[0], nil
}

func limitFirst(cond SelectBuilder) SelectBuilder {
	return cond.Offset(0).Limit(1)
}

type DBContext struct {
	db DBPgx
	tx pgx.Tx
}

type LogLevel int

const (
	Debug LogLevel = iota + 1
	Info
	Warn
	Error
)

func (ctx DBContext) logQuery(level LogLevel, msg string, err error, duration time.Duration, sql string, args ...interface{}) {
	if err != nil {
		log.Println("[\033[31mERROR\033[0m]\033[31m", msg, "\033[0m", "\n   | \033[32mError:\033[0m", err, "\n   | \033[32mArgs:\033[0m", args, "\n   | \033[32mSQL:\033[0m", sql)
		return
	}

	if level < Warn && duration > time.Millisecond*500 {
		level = Warn
		msg = "[SLOW QUERY] " + msg
	}

	switch level {
	case Debug:
		log.Println("[\033[35mDEBUG\033[0m]\033[33m", msg, "\033[0m", duration, "\n   | \033[32mArgs:\033[0m", args, "\n   | \033[32mSQL:\033[0m", sql)
	case Info:
		log.Println("[\033[34mINFO\033[0m]\033[33m", msg, "\033[0m", duration, "\n   | \033[32mArgs:\033[0m", args, "\n    | \033[32mSQL:\033[0m", sql)
	case Warn:
		log.Println("[\033[33mWARN\033[0m]\033[33m", msg, "\033[0m", duration, "\n   | \033[32mArgs:\033[0m", args, "\n    | \033[32mSQL:\033[0m", sql)
	case Error:
		log.Println("[\033[31mERROR\033[0m]\033[33m", msg, "\033[0m", duration, "\n   | \033[32mArgs:\033[0m", args, "\n   | \033[32mSQL:\033[0m", sql)
	}
}

type CustomQueries struct {
	ctx *DBContext
}

// Find TableNames records based on the provided conditions
//
// Usage:
//
//	entities, err := db.Queries.TableNames(
//	  // ... can use filters here (cf db.TableNames.Query.*)
//	)
func (q *CustomQueries) TableNames(filters ...WhereCondition) (requestData []TableNamesModel, requestErr error) {
	query := squirrel.Select("table_name")
	query = query.From("information_schema.columns").PlaceholderFormat(placeholder)
	query = query.Where("table_schema = 'public'")
	for _, filter := range filters {
		query = filter(query)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	defer func() {
		q.ctx.logQuery(Debug, "DB.Queries.TableNames", requestErr, time.Since(start), sql, args)
	}()

	return QueryMany[TableNamesModel](q.ctx, sql, args...)
}

type TableNamesModel struct {
	TableName *string `json:"table_name" db:"table_name"`
}

// Find TableColumns records based on the provided conditions
//
// Usage:
//
//	entities, err := db.Queries.TableColumns(
//	  // ... can use filters here (cf db.TableColumns.Query.*)
//	)
func (q *CustomQueries) TableColumns(filters ...WhereCondition) (requestData []TableColumnsModel, requestErr error) {
	query := squirrel.Select("column_name, column_default, is_nullable, data_type, udt_name")
	query = query.From("information_schema.columns").PlaceholderFormat(placeholder)
	query = query.Where("table_schema = 'public'")
	for _, filter := range filters {
		query = filter(query)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	defer func() {
		q.ctx.logQuery(Debug, "DB.Queries.TableColumns", requestErr, time.Since(start), sql, args)
	}()

	return QueryMany[TableColumnsModel](q.ctx, sql, args...)
}

type TableColumnsModel struct {
	ColumnName    *string `json:"column_name" db:"column_name"`
	ColumnDefault *string `json:"column_default" db:"column_default"`
	IsNullable    *string `json:"is_nullable" db:"is_nullable"`
	DataType      *string `json:"data_type" db:"data_type"`
	UdtName       *string `json:"udt_name" db:"udt_name"`
}
