package trader

import (
	httpClient "github.com/Artyom099/trading-bot/trader/internal/client/http"
	def "github.com/Artyom099/trading-bot/trader/internal/service"
)

var _ def.ITraderService = (*service)(nil)

type service struct {
	mt5Adapter    httpClient.IMT5AdapterClient
	chatID        int64
	defaultVolume float64
}

func NewService(mt5Adapter httpClient.IMT5AdapterClient, chatID int64, defaultVolume float64) *service {
	return &service{
		mt5Adapter:    mt5Adapter,
		chatID:        chatID,
		defaultVolume: defaultVolume,
	}
}
