package telegram

import "context"

// INotifier — отправка служебных сообщений в рабочий Telegram-чат.
type INotifier interface {
	Notify(ctx context.Context, text string) error
}
