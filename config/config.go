package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	localEnv      AppENV = "local"
	stagingEnv    AppENV = "stage"
	productionEnv AppENV = "prod"
)

var (
	appEnv  = AppENV(os.Getenv("APP_ENV"))
	cfg     Configuration
	cfgOnce sync.Once
)

type (
	AppENV string

	Configuration struct {
		App struct {
			Rest AppConfiguration
		}
		Store struct {
			Postgresql PGConfiguration
			Migration  MigrationConfiguration
			Redis      RedisConfiguration
		}
	}

	AppConfiguration struct {
		Enabled bool
		Name    string
		Port    int
	}

	StoreConfiguration struct {
		Password string
		Host     string
		Port     string
	}

	PGConfiguration struct {
		StoreConfiguration `mapstructure:",squash"`
		Name               string
		User               string
		SSLMode            string
	}

	RedisConfiguration struct {
		StoreConfiguration `mapstructure:",squash"`
		Database           int
	}

	MigrationConfiguration struct {
		TableName string
	}
)

func New() *Configuration {
	cfgOnce.Do(func() {
		if appEnv == "" {
			appEnv = localEnv
		}
		loadConfigYml(&cfg, "./config", fmt.Sprintf("config.%s", appEnv))
		viper.AutomaticEnv()
	})

	return &cfg
}

func loadConfigYml(cfg interface{}, path, name string) {
	filePath := fmt.Sprintf("%s/%s.yaml", path, name)
	if _, err := os.Stat(filePath); err != nil {
		return
	}

	viper.SetConfigName(name)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	if err := viper.ReadInConfig(); err != nil {
		panic(errors.Wrap(err, "fail read config"))
	}

	if err := viper.Unmarshal(cfg); err != nil {
		panic(errors.Wrap(err, "fail decode config"))
	}
}

func (d PGConfiguration) GetConfigString() string {
	if d.SSLMode == "" {
		d.SSLMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

func (a AppConfiguration) IsDebug() bool {
	return appEnv != productionEnv
}

func (r RedisConfiguration) GetConfigString() string {
	return fmt.Sprintf("%s:%v", r.Host, r.Port)
}
