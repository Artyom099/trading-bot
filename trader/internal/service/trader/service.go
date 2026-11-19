package trader

import (
	"time"

	httpClient "github.com/Artyom099/trading-bot/trader/internal/client/http"
	telegramClient "github.com/Artyom099/trading-bot/trader/internal/client/telegram"
	"github.com/Artyom099/trading-bot/trader/internal/killswitch"
	"github.com/Artyom099/trading-bot/trader/internal/model"
	def "github.com/Artyom099/trading-bot/trader/internal/service"
)

var _ def.ITraderService = (*service)(nil)

type service struct {
	mt5Adapter   httpClient.IMT5AdapterClient
	chatID       int64
	volumes      model.TPVolumes
	maxSignalAge time.Duration // 0 — фильтр по возрасту выключен
	killSwitch   *killswitch.Switch
	notifier     telegramClient.INotifier // nil — не уведомлять
}

func NewService(mt5Adapter httpClient.IMT5AdapterClient, chatID int64, volumes model.TPVolumes, maxSignalAge time.Duration, killSwitch *killswitch.Switch, notifier telegramClient.INotifier) *service {
	return &service{
		mt5Adapter:   mt5Adapter,
		chatID:       chatID,
		volumes:      volumes,
		maxSignalAge: maxSignalAge,
		killSwitch:   killSwitch,
		notifier:     notifier,
	}
}
