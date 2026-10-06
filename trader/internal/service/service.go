package service

import "context"

type ITraderService interface {
	HandleMessage(ctx context.Context, chatID int64, text string) error
}
