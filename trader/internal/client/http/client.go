package http

import (
	"context"

	"github.com/Artyom099/trading-bot/trader/internal/model"
)

// IMT5AdapterClient — HTTP-клиент к Python MT5 Adapter, не к брокеру.
type IMT5AdapterClient interface {
	OpenOrder(ctx context.Context, signal model.Signal) error
}
