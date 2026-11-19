package service

import (
	"context"
	"time"
)

type ITraderService interface {
	HandleMessage(ctx context.Context, chatID int64, text string, sentAt time.Time) error

	// HandleCommand обрабатывает команды управления (/stop, /start, /status) из рабочего чата.
	// handled=false — это не команда (или чат чужой), текст нужно разбирать как сигнал.
	// reply — что ответить в чат, пусто — не отвечать.
	HandleCommand(ctx context.Context, chatID int64, text string, sentAt time.Time) (reply string, handled bool)
}
