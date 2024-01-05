package rest_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gotinder/infra"
	"gotinder/rest"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pgTest     *postgresTest
	pgTestOnce sync.Once
)

type (
	httpTestBuilder struct {
		method string
		path   string
		body   io.Reader
		header http.Header
	}

	postgresTest struct {
		container *postgres.PostgresContainer
		connStr   string
		conn      *sql.DB
	}
)

func newHttpTest() *httpTestBuilder {
	gin.SetMode(gin.TestMode)
	h := make(http.Header)
	h.Add("Content-Type", "application/json")
	return &httpTestBuilder{
		method: http.MethodGet,
		header: h,
	}
}

func (b *httpTestBuilder) do() *http.Response {
	host := "localhost:3000"
	url := fmt.Sprintf("http://%s%s", host, b.path)
	request := httptest.NewRequest(b.method, url, b.body)
	request.Header = b.header
	recorder := httptest.NewRecorder()

	server := &http.Server{
		Handler:           rest.NewHandler(),
		ReadHeaderTimeout: 1 * time.Minute,
	}
	server.Handler.ServeHTTP(recorder, request)
	return recorder.Result()
}

func (b *httpTestBuilder) withMethod(method string) *httpTestBuilder {
	b.method = method
	return b
}

func (b *httpTestBuilder) withPath(path string) *httpTestBuilder {
	b.path = path
	return b
}

func (b *httpTestBuilder) withBody(body any) *httpTestBuilder {
	bodyJson, _ := json.Marshal(body)
	b.body = bytes.NewReader(bodyJson)
	return b
}

func newPostgresTest(t *testing.T) *postgresTest {
	pgTestOnce.Do(func() {
		var err error
		ctx := context.Background()
		pgTest = new(postgresTest)
		pgTest.container, err = postgres.RunContainer(
			ctx,
			testcontainers.WithImage("docker.io/postgis/postgis:15-3.4"),
			postgres.WithDatabase("test"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
		)
		require.NoError(t, err)

		pgTest.connStr, err = pgTest.container.ConnectionString(ctx, "sslmode=disable", "application_name=test")
		require.NoError(t, err)
		pgTest.conn, err = sql.Open("postgres", pgTest.connStr)
		require.NoError(t, err)
		require.NoError(t, pgTest.conn.Ping())

		infra.Migrate(fmt.Sprintf("%s&search_path=public", pgTest.connStr), "../migrations", "test_scheme_migrations")
	})
	return pgTest
}

func (p *postgresTest) migrate(t *testing.T, conn *sql.DB) {
	scheme := strings.ToLower(regexp.MustCompile(`\W`).ReplaceAllString(t.Name(), "_"))
	createSchema := fmt.Sprintf(`CREATE SCHEMA %s;`, scheme)
	_, err := conn.Exec(createSchema)
	require.NoError(t, err)

	setSchema := fmt.Sprintf(`SET search_path TO %s,public;`, scheme)
	_, err = conn.Exec(setSchema)
	require.NoError(t, err)

	infra.Migrate(fmt.Sprintf("%s&search_path=%s,public", p.connStr, scheme), "../migrations", "test_scheme_migrations")
}
