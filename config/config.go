package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Debug bool `env:"DEBUG" envDefault:"false"`

	Telegram struct {
		Token string `env:"TOKEN,required"`

		Webhook struct {
			URL  string `env:"URL"`
			Port int    `env:"PORT" envDefault:"3000"`
		}
	}

	Database struct {
		Path string `env:"PATH" envDefault:"./data/tasks.db"`
	}
}

func New() (*Config, error) {
	return env.ParseAs[*Config]()
}
