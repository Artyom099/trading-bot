# Python MT5 Adapter

Тонкий HTTP-сервис между Go `trader` и локальным терминалом MetaTrader 5.

До брокера адаптер **не ходит**. Он вызывает `order_send()` в уже запущенном `terminal64.exe` (Windows Named Pipe). Терминал сам отправляет ордер брокеру.

```text
trader  --HTTP-->  этот сервис  --MT5 Python API-->  terminal64.exe  -->  брокер
```

## Файлы

| Файл | Роль |
|---|---|
| `app.py` | Точка входа: HTTP (`MT5_ADAPTER_HOST`/`PORT`, по умолчанию `127.0.0.1:8080`) |
| `handler.py` | Маршруты: `GET /health`, `POST /v1/orders` |
| `order.py` | Разбор JSON от trader (`side`, `symbol`, `volume`, `sl`, `tp`) |
| `terminal.py` | `initialize` / `order_send` / `shutdown`. Нет MT5 — режим dry-run |

## API

`POST /v1/orders`

```json
{"side": "sell", "symbol": "XAUUSD", "volume": 0.01, "sl": 4175.89, "tp": 4163.38}
```

- `200` — ордер принят (`mode: live` или `dry-run`)
- `400` — кривое тело
- `502` — терминал отклонил заявку

`GET /health` → `{"status": "ok", "live": true/false}`

## Live vs dry-run

- **live** — Windows, установлен `MetaTrader5`, открыт терминал, включена автоторговля, `initialize()` успешен.
- **dry-run** — пакет не установлен или терминал не поднялся. Ордер только логируется, в рынок не уходит.

На Mac/Linux адаптер стартует, но всегда в dry-run: официальный Python API — Windows-only.

## Запуск

```bash
pip3 install -r python/mt5_adapter/requirements.txt
python3 python/mt5_adapter/app.py
```

Либо `task run:mt5-adapter`.
