package trader

import (
	"context"

	"go.uber.org/zap"

	"github.com/Artyom099/trading-bot/platform/pkg/logger"
	telegram_v1 "github.com/Artyom099/trading-bot/shared/pkg/telegram/v1"
	telegramConverter "github.com/Artyom099/trading-bot/trader/internal/converter/telegram"
)

func (s *service) HandleMessage(ctx context.Context, chatID int64, text string) error {
	if chatID != s.chatID {
		return nil
	}

	in, err := telegram_v1.Decode(text)
	if err != nil {
		logger.Warn(ctx, "skip telegram message", zap.String("text", text), zap.Error(err))
		return nil
	}

	signal := telegramConverter.ToModel(in)
	signal.Volume = s.defaultVolume

	logger.Info(ctx, "opening trade",
		zap.String("side", string(signal.Side)),
		zap.String("symbol", signal.Symbol),
		zap.Float64("volume", signal.Volume),
		zap.Float64("entry", signal.Entry),
		zap.Float64("sl", signal.SL),
		zap.Float64("tp", signal.TP),
	)

	return s.mt5Adapter.OpenOrder(ctx, signal)
}
