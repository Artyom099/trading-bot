package env

import env "github.com/caarlos0/env/v11"

type mt5EnvConfig struct {
	AdapterURL string `env:"MT5_ADAPTER_URL,required"`
}

type mt5Config struct {
	raw mt5EnvConfig
}

func NewMT5Config() (*mt5Config, error) {
	var raw mt5EnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &mt5Config{raw: raw}, nil
}

func (cfg *mt5Config) AdapterURL() string {
	return cfg.raw.AdapterURL
}
