package main

import (
	"gotinder/config"
	"gotinder/rest"
)

func main() {
	cfg := config.New()
	if cfg.App.Rest.Enabled {
		rest.New(cfg.App.Rest.Port)
	}
}
