package main

import (
	"gotinder/config"
	"gotinder/infra"
	"gotinder/rest"

	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.New()
	infra.NewPgConnection(cfg.Store.Postgresql.GetConfigString())
	infra.Migrate(cfg.Store.Postgresql.GetConfigString(), "./migrations", cfg.Store.Migration.TableName)
	infra.NewRedisPool(cfg.Store.Redis.GetConfigString(), cfg.Store.Redis.Password, cfg.Store.Redis.Database)
	if cfg.App.Rest.Enabled {
		rest.New(
			cfg.App.Rest.Port,
			func() (name string, fn func()) {
				return "close pg connection", infra.TerminatePgConnection
			},
			func() (name string, fn func()) {
				return "close redis connection", infra.TerminalRedisPool
			},
		)
	}
}
