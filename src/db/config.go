package db

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type DbConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	SslMode  string `yaml:"sslMode"`
}

func LoadDBConfig(path string) (*DbConfig, error) {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	var cfg DbConfig
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
