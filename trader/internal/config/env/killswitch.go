package env

import (
	"fmt"

	env "github.com/caarlos0/env/v11"
)

type killSwitchEnvConfig struct {
	MaxConsecutiveErrors int    `env:"KILLSWITCH_MAX_CONSECUTIVE_ERRORS" envDefault:"5"`
	MaxSignalsPerDay     int    `env:"KILLSWITCH_MAX_SIGNALS_PER_DAY" envDefault:"20"`
	StateFile            string `env:"KILLSWITCH_STATE_FILE" envDefault:"../deploy/compose/trader/killswitch.json"`
}

type killSwitchConfig struct {
	raw killSwitchEnvConfig
}

func NewKillSwitchConfig() (*killSwitchConfig, error) {
	var raw killSwitchEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	if raw.MaxConsecutiveErrors < 0 {
		return nil, fmt.Errorf("KILLSWITCH_MAX_CONSECUTIVE_ERRORS must be >= 0, got %d", raw.MaxConsecutiveErrors)
	}
	if raw.MaxSignalsPerDay < 0 {
		return nil, fmt.Errorf("KILLSWITCH_MAX_SIGNALS_PER_DAY must be >= 0, got %d", raw.MaxSignalsPerDay)
	}

	return &killSwitchConfig{raw: raw}, nil
}

// MaxConsecutiveErrors — стоп после N неудачных сделок подряд. 0 — не проверять.
func (cfg *killSwitchConfig) MaxConsecutiveErrors() int {
	return cfg.raw.MaxConsecutiveErrors
}

// MaxSignalsPerDay — стоп, когда за 24 часа принято больше N сигналов. 0 — не проверять.
func (cfg *killSwitchConfig) MaxSignalsPerDay() int {
	return cfg.raw.MaxSignalsPerDay
}

// StateFile — файл, в котором хранится состояние «остановлено».
func (cfg *killSwitchConfig) StateFile() string {
	return cfg.raw.StateFile
}
