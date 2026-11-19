package config

import (
	"time"

	"github.com/Artyom099/trading-bot/trader/internal/model"
)

type LoggerConfig interface {
	Level() string
	AsJson() bool
	EnableOTLP() bool
}

type TelegramConfig interface {
	Token() string
	ChatID() int64
}

type MT5Config interface {
	AdapterURL() string
}

type KillSwitchConfig interface {
	MaxConsecutiveErrors() int
	MaxSignalsPerDay() int
	StateFile() string
}

type TradeConfig interface {
	Volumes() model.TPVolumes
	MaxSignalAge() time.Duration
}
