package telegram_v1

import "errors"

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// Signal — входящее сообщение Telegram по контракту v1.
// Volume в контракт не входит: его задаёт trader из TRADE_VOLUME.
type Signal struct {
	Side   Side
	Symbol string
	SL     float64
	TP1    float64
	TP2    float64
	TP3    float64
	Entry  float64
}

var (
	ErrMissingSide   = errors.New("telegram_v1: missing side")
	ErrMissingSymbol = errors.New("telegram_v1: missing symbol")
	ErrMissingSL     = errors.New("telegram_v1: missing sl")
)
