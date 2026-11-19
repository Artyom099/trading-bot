package telegram

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
)

const sendTimeout = 5 * time.Second

type notifier struct {
	bot    *bot.Bot
	chatID int64
}

func NewNotifier(b *bot.Bot, chatID int64) *notifier {
	return &notifier{bot: b, chatID: chatID}
}

func (n *notifier) Notify(ctx context.Context, text string) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	_, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: n.chatID, Text: text})

	return err
}
