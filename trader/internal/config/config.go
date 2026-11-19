package config

import (
	"os"

	"github.com/joho/godotenv"

	"github.com/Artyom099/trading-bot/trader/internal/config/env"
)

var appConfig *config

type config struct {
	Logger     LoggerConfig
	Telegram   TelegramConfig
	MT5        MT5Config
	Trade      TradeConfig
	KillSwitch KillSwitchConfig
}

func Load(path ...string) error {
	err := godotenv.Load(path...)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	loggerCfg, err := env.NewLoggerConfig()
	if err != nil {
		return err
	}

	telegramCfg, err := env.NewTelegramConfig()
	if err != nil {
		return err
	}

	mt5Cfg, err := env.NewMT5Config()
	if err != nil {
		return err
	}

	tradeCfg, err := env.NewTradeConfig()
	if err != nil {
		return err
	}

	killSwitchCfg, err := env.NewKillSwitchConfig()
	if err != nil {
		return err
	}

	appConfig = &config{
		Logger:     loggerCfg,
		Telegram:   telegramCfg,
		MT5:        mt5Cfg,
		Trade:      tradeCfg,
		KillSwitch: killSwitchCfg,
	}

	return nil
}

func AppConfig() *config {
	return appConfig
}
