package app

import (
	"fmt"

	"github.com/go-telegram/bot"

	httpClient "github.com/Artyom099/trading-bot/trader/internal/client/http"
	mt5adapter "github.com/Artyom099/trading-bot/trader/internal/client/http/mt5adapter"
	"github.com/Artyom099/trading-bot/trader/internal/config"
	"github.com/Artyom099/trading-bot/trader/internal/service"
	traderService "github.com/Artyom099/trading-bot/trader/internal/service/trader"
)

type diContainer struct {
	traderService    service.ITraderService
	mt5AdapterClient httpClient.IMT5AdapterClient
	telegramBot      *bot.Bot
}

func NewDiContainer() *diContainer {
	return &diContainer{}
}

func (d *diContainer) TraderService() service.ITraderService {
	if d.traderService == nil {
		d.traderService = traderService.NewService(
			d.MT5AdapterClient(),
			config.AppConfig().Telegram.ChatID(),
			config.AppConfig().Trade.Volume(),
		)
	}

	return d.traderService
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
