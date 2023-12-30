package infra

import (
	"database/sql"
	"log"
	"net/url"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	"github.com/pkg/errors"
)

var (
	pgConn     *sql.DB
	pgConnOnce sync.Once
)

func NewPgConnection(connStr string) {
	pgConnOnce.Do(func() {
		conn, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(errors.Wrap(err, "fail to open connection"))
		}

		err = conn.Ping()
		if err != nil {
			panic(errors.Wrap(err, "fail to verify connection"))
		}

		conn.SetMaxIdleConns(1)
		conn.SetMaxOpenConns(4)
		conn.SetConnMaxLifetime(3600 * time.Second)

		log.Println("database connection established!")

		pgConn = conn
	})
}

func TerminatePgConnection() {
	defer func() {
		pgConn = nil
		pgConnOnce = sync.Once{}
	}()
	if pgConn == nil {
		return
	}
	log.Println("clossing db connection")
	if err := pgConn.Close(); err != nil {
		log.Println(errors.Wrap(err, "failed to close pg connection"))
	} else {
		log.Println("db connection closed")
	}
}

func Migrate(connStr, migrationDir, migrationTableName string) {
	u, err := url.Parse(connStr)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse config"))
	}
	db := dbmate.New(u)
	db.MigrationsDir = []string{migrationDir}
	db.MigrationsTableName = migrationTableName

	log.Println("apply pending migration")
	err = db.CreateAndMigrate()
	if err != nil {
		panic(errors.Wrap(err, "failed to migrate"))
	}
}
