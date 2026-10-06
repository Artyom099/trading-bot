module github.com/Artyom099/trading-bot/trader

go 1.27.1

replace github.com/Artyom099/trading-bot/platform => ../platform

replace github.com/Artyom099/trading-bot/shared => ../shared

require (
	github.com/Artyom099/trading-bot/platform v0.0.0-00010101000000-000000000000
	github.com/Artyom099/trading-bot/shared v0.0.0-00010101000000-000000000000
	github.com/caarlos0/env/v11 v11.4.1
	github.com/go-telegram/bot v1.27.0
	github.com/joho/godotenv v1.5.1
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.10.0 // indirect
