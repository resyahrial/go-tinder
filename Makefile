GOCMD=go
GOTEST=$(GOCMD) test
GOPATH=$(shell $(GOCMD) env GOPATH)
AIRPATH=$(GOPATH)/bin/air

.PHONY: check-db-env migratedown migratenew migrateup lint sqlc test watch tidy

check-db-env:
ifndef PSQL_DB_NAME
	$(error PSQL_DB_NAME is undefined)
endif

migratedown: check-db-env
	dbmate -d ./migrations -u "postgres://${PSQL_USER}:${PSQL_PWD}@${PSQL_HOST}:${PSQL_PORT}/${PSQL_DB_NAME}?sslmode=$(if $(PSQL_SSL_MODE),$(PSQL_SSL_MODE),disable)" down

migratenew:
	dbmate -d ./migrations new ${name}

migrateup: check-db-env
	dbmate -d ./migrations -u "postgres://${PSQL_USER}:${PSQL_PWD}@${PSQL_HOST}:${PSQL_PORT}/${PSQL_DB_NAME}?sslmode=$(if $(PSQL_SSL_MODE),$(PSQL_SSL_MODE),disable)" up

sqlc:
	sqlc generate

test:
	GOARCH=amd64 $(GOTEST) -v -race ./...

watch:
	make tidy
	test -s ${AIRPATH} || curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(GOPATH)/bin
	GO_APP_ENV=local APP_VERSION=local ${AIRPATH}

tidy:
	$(GOCMD) mod tidy

lint:
	golangci-lint run ${dir}
