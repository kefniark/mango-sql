package introspection

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"testing"

	introspect "github.com/kefniark/mango-sql/internal/database/postgres"
	"github.com/kefniark/mango-sql/internal/generator"
	"github.com/kefniark/mango-sql/tests/helpers"
	"github.com/stretchr/testify/assert"
)

//go:embed *.sql
var sqlFS embed.FS

func newTestDB(t *testing.T) string {
	data, err := sqlFS.ReadFile("seed.sql")
	if err != nil {
		panic(err)
	}

	config := helpers.NewDBConfigWith(t, data, "postgres.introspection")

	return config.URL()
}

func TestIntrospection(t *testing.T) {
	url := newTestDB(t)

	schema, err := introspect.Parse(url)
	assert.NoError(t, err)

	fmt.Println("Got", schema)

	var b bytes.Buffer
	contents := bufio.NewWriter(&b)

	err = generator.Generate(schema, contents, "introspection", "pgx", "none")
	assert.NoError(t, err)
	assert.NoError(t, contents.Flush())

	f, err := os.Create("client.go")
	assert.NoError(t, err)

	defer f.Close()

	_, err = f.Write(b.Bytes())
	assert.NoError(t, err)
}
