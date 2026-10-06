package model

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Signal struct {
	Side   Side    // направление сделки: buy или sell
	Symbol string  // тикер MT5, например XAUUSD
	Volume float64 // объём лота из TRADE_VOLUME, не из Telegram
	Entry  float64 // зона входа из сигнала; при открытии ордера пока не используется
	SL     float64 // обязательный стоп
	TP     float64 // первая цель (TP1), уходит в ордер MT5 как tp
	TP2    float64 // вторая цель; парсится, в ордер пока не ставится
	TP3    float64 // третья цель; парсится, в ордер пока не ставится
}
