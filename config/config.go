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
		Path string `env:"DB_PATH" envDefault:"./data/tasks.db"`
	}
}

func New() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	return &cfg, err
}
