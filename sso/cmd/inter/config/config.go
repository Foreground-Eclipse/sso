package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string        `yaml:"env" env-default:"local"`
	StoragePath string        `yaml:"storage_path" env-required:"true"`
	TokenTTL    time.Duration `yaml:"token_ttl" env-required:"true"`
	GRPC        GRPCConfig    `yaml:"grpc"`
}

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// MustLoad panics and doesnt return any error if theres any it will just panic.
func MustLoad() *Config {
	path := fetchConfigPath()
	if path == "" {
		panic("Config path is empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic("Config file does not exist")
	}

	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		panic("failed to read config " + err.Error())
	}

	return &cfg
}

func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to the config file")
	flag.Parse()
	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	return res
}
