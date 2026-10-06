package telegram

import (
	telegram_v1 "github.com/Artyom099/trading-bot/shared/pkg/telegram/v1"
	"github.com/Artyom099/trading-bot/trader/internal/model"
)

func ToModel(in telegram_v1.Signal) model.Signal {
	return model.Signal{
		Side:   model.Side(in.Side),
		Symbol: in.Symbol,
		SL:     in.SL,
		TP:     in.TP1,
		TP2:    in.TP2,
		TP3:    in.TP3,
		Entry:  in.Entry,
	}
}
