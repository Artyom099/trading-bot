# trading-bot

## Services
```mermaid
flowchart LR
    Telegram[Telegram chat] -->|Telegram Bot API HTTPS| Trader[Trader Service]
    Trader -->|HTTP| Adapter[Python MT5 Adapter]
    Adapter -->|MT5 Python API / IPC| MT5[MT5 Terminal]
    MT5 --> Broker[Broker]
```

## Quick start
1. Скопировать `deploy/compose/trader/.env.example` в `.env` и заполнить (для адаптера аналогично: `python/mt5_adapter/.env.example`)
2. `task run:mt5-adapter` — Python-адаптер на Windows с открытым MT5
3. `task run:trader` — Go-сервис слушает один Telegram-чат и открывает сделки

## Commands

- Run unit-tests:
```
task test:unit
```

- Format:
```
task format
```

- Lint:
```
task lint:do
```
