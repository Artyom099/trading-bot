package app

import (
	"fmt"

	"github.com/go-telegram/bot"

	httpClient "github.com/Artyom099/trading-bot/trader/internal/client/http"
	mt5adapter "github.com/Artyom099/trading-bot/trader/internal/client/http/mt5adapter"
	telegramClient "github.com/Artyom099/trading-bot/trader/internal/client/telegram"
	"github.com/Artyom099/trading-bot/trader/internal/config"
	"github.com/Artyom099/trading-bot/trader/internal/killswitch"
	"github.com/Artyom099/trading-bot/trader/internal/service"
	traderService "github.com/Artyom099/trading-bot/trader/internal/service/trader"
)

type diContainer struct {
	traderService    service.ITraderService
	mt5AdapterClient httpClient.IMT5AdapterClient
	telegramBot      *bot.Bot
	killSwitch       *killswitch.Switch
}

func NewDiContainer() *diContainer {
	return &diContainer{}
}

func (d *diContainer) TraderService() service.ITraderService {
	if d.traderService == nil {
		d.traderService = traderService.NewService(
			d.MT5AdapterClient(),
			config.AppConfig().Telegram.ChatID(),
			config.AppConfig().Trade.Volumes(),
			config.AppConfig().Trade.MaxSignalAge(),
			d.killSwitch,
			telegramClient.NewNotifier(d.TelegramBot(), config.AppConfig().Telegram.ChatID()),
		)
	}

	return d.traderService
}

// InitKillSwitch читает сохранённое состояние стоп-крана. Вызывать до TraderService().
func (d *diContainer) InitKillSwitch() error {
	ks, err := killswitch.New(killswitch.Config{
		MaxConsecutiveErrors: config.AppConfig().KillSwitch.MaxConsecutiveErrors(),
		MaxSignalsPerDay:     config.AppConfig().KillSwitch.MaxSignalsPerDay(),
		StateFile:            config.AppConfig().KillSwitch.StateFile(),
	})
	if err != nil {
		return err
	}

	d.killSwitch = ks

	return nil
}

func (d *diContainer) MT5AdapterClient() httpClient.IMT5AdapterClient {
	if d.mt5AdapterClient == nil {
		d.mt5AdapterClient = mt5adapter.NewClient(config.AppConfig().MT5.AdapterURL())
	}

	return d.mt5AdapterClient
}

func (d *diContainer) TelegramBot() *bot.Bot {
	if d.telegramBot == nil {
		b, err := bot.New(
			config.AppConfig().Telegram.Token(),
			bot.WithAllowedUpdates(bot.AllowedUpdates{"message", "channel_post"}),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create telegram bot: %s\n", err.Error()))
		}

		d.telegramBot = b
	}

	return d.telegramBot
}
