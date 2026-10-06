package env

import env "github.com/caarlos0/env/v11"

type tradeEnvConfig struct {
	Volume float64 `env:"TRADE_VOLUME,required"`
}

type tradeConfig struct {
	raw tradeEnvConfig
}

func NewTradeConfig() (*tradeConfig, error) {
	var raw tradeEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &tradeConfig{raw: raw}, nil
}

func (cfg *tradeConfig) Volume() float64 {
	return cfg.raw.Volume
}
