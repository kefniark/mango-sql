package autoincrement

import (
	"context"
	"embed"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
)

//go:generate go run ../../../cmd/mangosql/ --output client.go --package autoincrement --logger console ./schema.sql

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) (*DBClient, func()) {
	data, err := sqlFS.ReadFile("schema.sql")
	if err != nil {
		panic(err)
	}

	config := helpers.NewDBConfigWith(t, data, "postgres.auto-increment")
	db, err := pgx.Connect(context.Background(), config.URL())
	if err != nil {
		panic(err)
	}

	return New(db), func() {
		db.Close(context.Background())
	}
}

func TestAutoIncrement(t *testing.T) {
	db, close := newTestDB(t)
	defer close()

	input := CompanyCreate{
		Name:    "BobCorp",
		Age:     15,
		Address: "15th blv avenue",
		Salary:  12.5,
	}

	company, err := db.Company.Insert(input)
	assert.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(company.Name), input.Name)
	assert.Equal(t, company.Age, input.Age)
	assert.Equal(t, strings.TrimSpace(company.Address), input.Address)
	assert.Equal(t, company.Salary, input.Salary)
}
