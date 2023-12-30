package main

import (
	"gotinder/config"
	"gotinder/infra"
	"gotinder/rest"
)

func main() {
	cfg := config.New()
	infra.NewPgConnection(cfg.Store.Postgresql.GetConfigString())
	infra.Migrate(cfg.Store.Postgresql.GetConfigString(), "./migrations", cfg.Store.Migration.TableName)
	if cfg.App.Rest.Enabled {
		rest.New(
			cfg.App.Rest.Port,
			func() (name string, fn func()) {
				return "close pg connection", infra.TerminatePgConnection
			},
		)
	}
}
