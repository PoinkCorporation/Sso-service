package config

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var (
	configOnce  sync.Once
	configValue string
)

const defaultPathToConfigFile = "/home/maksim/Projects/Go/Project/sso/config/config_local.yaml"

type Config struct {
	Env            string     `yaml:"env" env-default:"local"`
	StoragePath    string     `yaml:"storage_path" env-required:"true"`
	GRPC           GRPCConfig `yaml:"grpc"`
	MigrationsPath string
	TokenTTL       time.Duration `yaml:"token_ttl" env-default:"1h"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		configPath = defaultPathToConfigFile
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("config path is empty: " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	configOnce.Do(func() {
		flag.StringVar(&configValue, "config", "", "path to config file")
		flag.Parse()
		if configValue == "" {
			configValue = os.Getenv("CONFIG_PATH")
		}
	})
	return configValue
}
