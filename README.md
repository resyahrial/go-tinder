# Go-Tinder

Go-Tinder is a simple Dating Apps inspired by popular apps like Tinder/Bumble.
Current feature:

* Register and Login
* Update current location
* Get user recommendations
* Doing action (like or pass)
* Apply as subscribed user

## Run locally

Make sure `docker` installed on your machine.

1. Duplicate `config.example.yaml` file
2. Rename to `config.stage.yaml`
3. Run `docker-compose up -d`
4. App can be accessed on `localhost:8080`

## Structure

### Config

Contain all configuration for the app

### Infra

Contain the implementation of used infrastructure (Postgresql and Redis)

### Migrations

Contain migration scripts

### Rest

Contain implementation of Rest API and Business Logic

## Other function

* `make test` to run integration test. don't bother to prepare the infrastructure, this project use [`testcontainers`](https://golang.testcontainers.org/) to provide it. make sure `docker` is active.
* `make lint` to lint the code. this project use [`golangci-lint`](https://golangci-lint.run/).
