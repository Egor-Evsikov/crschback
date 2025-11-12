package server

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type ServerConfig struct {
	Addr     string `yaml:"addr" env-default:":8080"`
	LogLever string `yaml:"loglevel" env-default:"debug"`
}

func LoadServerConfig(path string) (*ServerConfig, error) {

	var cfg ServerConfig
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
