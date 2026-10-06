package config

type LoggerConfig interface {
	Level() string
	AsJson() bool
	EnableOTLP() bool
}

type TelegramConfig interface {
	Token() string
	ChatID() int64
}

type MT5Config interface {
	AdapterURL() string
}

type TradeConfig interface {
	Volume() float64
}
