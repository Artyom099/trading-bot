package model

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Signal struct {
	Side   Side    // направление сделки: buy или sell
	Symbol string  // тикер MT5, например XAUUSD
	Volume float64 // объём лота из TRADE_VOLUME_TP*, не из Telegram
	Entry  float64 // зона входа из сигнала; при открытии ордера пока не используется
	SL     float64 // обязательный стоп
	TP     float64 // первая цель (TP1), уходит в ордер MT5 как tp
	TP2    float64 // вторая цель (своя сделка)
	TP3    float64 // третья цель (своя сделка)

	Comment string // метка сделки для MT5 (tp1/tp2/tp3); в Telegram не приходит
}

// TPVolumes — объём лота для сделки на каждую цель. 0 — сделку на эту цель не открываем.
type TPVolumes struct {
	TP1 float64
	TP2 float64
	TP3 float64
}
