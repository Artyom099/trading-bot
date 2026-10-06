package app

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/Artyom099/trading-bot/platform/pkg/closer"
	"github.com/Artyom099/trading-bot/platform/pkg/logger"
	"github.com/Artyom099/trading-bot/trader/internal/config"
)

type App struct {
	diContainer *diContainer
}

func New(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	go a.runTelegramBot(ctx)

	<-ctx.Done()
	logger.Info(ctx, "Shutdown signal received")
	return nil
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initDI,
		a.initLogger,
		a.initCloser,
		a.initTelegramBot,
	}

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initDI(_ context.Context) error {
	a.diContainer = NewDiContainer()
	return nil
}

func (a *App) initLogger(ctx context.Context) error {
	return logger.Init(
		ctx,
		config.AppConfig().Logger.Level(),
		config.AppConfig().Logger.AsJson(),
		config.AppConfig().Logger.EnableOTLP(),
	)
}

func (a *App) initCloser(_ context.Context) error {
	closer.SetLogger(logger.Logger())
	return nil
}

func (a *App) initTelegramBot(_ context.Context) error {
	telegramBot := a.diContainer.TelegramBot()

	telegramBot.RegisterHandlerMatchFunc(func(_ *models.Update) bool {
		return true
	}, a.handleTelegramUpdate)

	return nil
}

func (a *App) handleTelegramUpdate(ctx context.Context, _ *bot.Bot, update *models.Update) {
	msg := update.Message
	if msg == nil {
		msg = update.ChannelPost
	}
	if msg == nil || msg.Text == "" {
		return
	}

	err := a.diContainer.TraderService().HandleMessage(ctx, msg.Chat.ID, msg.Text)
	if err != nil {
		logger.Error(ctx, "failed to handle telegram message", zap.Error(err))
	}
}

func (a *App) runTelegramBot(ctx context.Context) {
	logger.Info(ctx, "🚀 Telegram bot running")
	a.diContainer.TelegramBot().Start(ctx)
}
