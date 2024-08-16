package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sync"
)

type (
	Config struct {
		TelegramBot `yaml:"telegramBot"`
		Postgres    `yaml:"postgres"`
	}

	TelegramBot struct {
		Token string `yaml:"token"`
	}

	Postgres struct {
		ConnString string `yaml:"connString"`
	}
)

var (
	once   sync.Once
	config *Config
	err    error
)

func New(cfgPath string) (*Config, error) {
	once.Do(func() {
		config = &Config{}
		file, errOpen := os.Open(cfgPath)
		fmt.Print(file)
		if err != nil {
			err = errOpen
			return
		}
		defer file.Close()

		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return
		}
	})
	return config, err

}
