package env

import (
	"fmt"
	"time"

	env "github.com/caarlos0/env/v11"

	"github.com/Artyom099/trading-bot/trader/internal/model"
)

type tradeEnvConfig struct {
	VolumeTP1    float64       `env:"TRADE_VOLUME_TP1,required"`
	VolumeTP2    float64       `env:"TRADE_VOLUME_TP2,required"`
	VolumeTP3    float64       `env:"TRADE_VOLUME_TP3,required"`
	MaxSignalAge time.Duration `env:"TRADE_MAX_SIGNAL_AGE" envDefault:"60s"`
}

type tradeConfig struct {
	raw tradeEnvConfig
}

func NewTradeConfig() (*tradeConfig, error) {
	var raw tradeEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	for name, v := range map[string]float64{
		"TRADE_VOLUME_TP1": raw.VolumeTP1,
		"TRADE_VOLUME_TP2": raw.VolumeTP2,
		"TRADE_VOLUME_TP3": raw.VolumeTP3,
	} {
		if v < 0 {
			return nil, fmt.Errorf("%s must be >= 0, got %v", name, v)
		}
	}

	return &tradeConfig{raw: raw}, nil
}

// Volumes — объём лота на каждую цель. 0 — сделку на эту цель не открываем.
func (cfg *tradeConfig) Volumes() model.TPVolumes {
	return model.TPVolumes{
		TP1: cfg.raw.VolumeTP1,
		TP2: cfg.raw.VolumeTP2,
		TP3: cfg.raw.VolumeTP3,
	}
}

// MaxSignalAge — сообщения старше этого возраста пропускаются. 0 — не фильтровать.
func (cfg *tradeConfig) MaxSignalAge() time.Duration {
	return cfg.raw.MaxSignalAge
}
