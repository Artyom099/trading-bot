package trader

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/Artyom099/trading-bot/platform/pkg/logger"
	telegram_v1 "github.com/Artyom099/trading-bot/shared/pkg/telegram/v1"
	telegramConverter "github.com/Artyom099/trading-bot/trader/internal/converter/telegram"
	"github.com/Artyom099/trading-bot/trader/internal/model"
)

func (s *service) HandleMessage(ctx context.Context, chatID int64, text string, sentAt time.Time) error {
	if chatID != s.chatID {
		return nil
	}

	if age := time.Since(sentAt); s.maxSignalAge > 0 && age > s.maxSignalAge {
		logger.Warn(ctx, "skip stale telegram message",
			zap.String("text", text),
			zap.Duration("age", age.Round(time.Second)),
			zap.Duration("max_age", s.maxSignalAge),
		)
		return nil
	}

	in, err := telegram_v1.Decode(text)
	if err != nil {
		logger.Warn(ctx, "skip telegram message", zap.String("text", text), zap.Error(err))
		return nil
	}

	signal := telegramConverter.ToModel(in)

	trades := s.buildTrades(signal)
	if len(trades) == 0 {
		logger.Warn(ctx, "no trades to open: no targets in signal or zero volume for them", zap.String("text", text))
		return nil
	}

	allowed, stoppedNow, err := s.killSwitch.AllowSignal()
	if err != nil {
		logger.Error(ctx, "failed to persist kill switch state", zap.Error(err))
	}
	if stoppedNow {
		s.onAutoStop(ctx, "Этот сигнал пропущен.")
	}
	if !allowed {
		_, reason := s.killSwitch.Stopped()
		logger.Warn(ctx, "trading is stopped, signal skipped", zap.String("reason", reason), zap.String("text", text))
		return nil
	}

	// Сделки открываем по очереди. Сбой одной не отменяет остальные, повторов нет:
	// после таймаута неизвестно, открылась ли сделка, и повтор может дать дубль.
	var errs []error
	for _, trade := range trades {
		if stopped, reason := s.killSwitch.Stopped(); stopped {
			logger.Warn(ctx, "trading stopped, remaining trades skipped",
				zap.String("target", trade.Comment), zap.String("reason", reason))
			break
		}

		logger.Info(ctx, "opening trade",
			zap.String("target", trade.Comment),
			zap.String("side", string(trade.Side)),
			zap.String("symbol", trade.Symbol),
			zap.Float64("volume", trade.Volume),
			zap.Float64("entry", trade.Entry),
			zap.Float64("sl", trade.SL),
			zap.Float64("tp", trade.TP),
		)

		openErr := s.mt5Adapter.OpenOrder(ctx, trade)
		if openErr != nil {
			logger.Error(ctx, "failed to open trade", zap.String("target", trade.Comment), zap.Error(openErr))
			errs = append(errs, fmt.Errorf("%s: %w", trade.Comment, openErr))
		}

		stoppedNow, saveErr := s.killSwitch.RecordTradeResult(openErr)
		if saveErr != nil {
			logger.Error(ctx, "failed to persist kill switch state", zap.Error(saveErr))
		}
		if stoppedNow {
			s.onAutoStop(ctx, lastErrorNote(openErr))
		}
	}

	return errors.Join(errs...)
}

// buildTrades делает по одной сделке на каждую цель сигнала (TP1, TP2, TP3) с её объёмом.
// Цель, которой нет в сигнале, или с нулевым объёмом пропускается.
// Если в сигнале нет ни одной цели, ничего не открывается.
func (s *service) buildTrades(signal model.Signal) []model.Signal {
	targets := []struct {
		name   string
		price  float64
		volume float64
	}{
		{"tp1", signal.TP, s.volumes.TP1},
		{"tp2", signal.TP2, s.volumes.TP2},
		{"tp3", signal.TP3, s.volumes.TP3},
	}

	var trades []model.Signal
	for _, t := range targets {
		if t.price <= 0 || t.volume <= 0 {
			continue
		}

		trades = append(trades, tradeFor(signal, t.name, t.price, t.volume))
	}

	return trades
}

func tradeFor(signal model.Signal, name string, tp, volume float64) model.Signal {
	signal.TP = tp
	signal.TP2 = 0
	signal.TP3 = 0
	signal.Volume = volume
	signal.Comment = name

	return signal
}

const maxErrorNoteLen = 300

// onAutoStop пишет в лог и в Telegram, что торговлю остановил автостоп, и почему.
// Сбой отправки в Telegram только логируется: остановка уже произошла.
func (s *service) onAutoStop(ctx context.Context, note string) {
	_, reason := s.killSwitch.Stopped()
	logger.Error(ctx, "⛔ kill switch triggered: trading stopped, send /start to resume", zap.String("reason", reason))

	if s.notifier == nil {
		return
	}

	text := fmt.Sprintf("⛔ Автостоп: %s.\nНовые сделки не открываются, открытые позиции не тронуты.", reason)
	if note != "" {
		text += "\n" + note
	}
	text += "\nВключить: /start"

	if err := s.notifier.Notify(ctx, text); err != nil {
		logger.Error(ctx, "failed to send kill switch notification", zap.Error(err))
	}
}

func lastErrorNote(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	if runes := []rune(msg); len(runes) > maxErrorNoteLen {
		msg = string(runes[:maxErrorNoteLen]) + "…"
	}

	return "Последняя ошибка: " + msg
}
