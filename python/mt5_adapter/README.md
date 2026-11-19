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
{
    "side": "sell", 
    "symbol": "XAUUSD", 
    "volume": 0.01, 
    "sl": 4175.89, 
    "tp": 4163.38
}
```

- `200` — ордер принят (`mode: live` или `dry-run`)
- `400` — кривое тело
- `502` — терминал отклонил заявку

`GET /health` →

```json
{
    "status": "ok", 
    "live": true, 
    "account": {
        "login": 123, 
        "server": "Broker-Demo", 
        "mode": "demo"
    }, 
    "allowed_account_mode": "demo"
}
```

`account` — `null` в dry-run.

## Защита от неверного счёта

Адаптер сверяет тип счёта в терминале (`demo` / `real`) с `MT5_ALLOWED_ACCOUNT_MODE` перед каждым ордером. По умолчанию `demo`: если в терминале залогинен реальный счёт, ордер отклоняется (`502`, `account mode real ... is not allowed`). Для реальной торговли нужно осознанно задать `MT5_ALLOWED_ACCOUNT_MODE=real`. Если счёт проверить нельзя, ордер тоже отклоняется. В dry-run проверки нет.

## Live vs dry-run

- **live** — Windows, установлен `MetaTrader5`, открыт терминал, включена автоторговля, `initialize()` успешен.
- **dry-run** — пакет не установлен или терминал не поднялся. Ордер только логируется, в рынок не уходит.

На Mac/Linux адаптер стартует, но всегда в dry-run: официальный Python API — Windows-only.

## Настройки

Читаются из `.env` рядом с `app.py` (шаблон: `.env.example`) и из переменных окружения процесса. Переменные окружения приоритетнее файла. Нет файла — используются значения по умолчанию.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `MT5_ADAPTER_HOST` | `127.0.0.1` | Адрес HTTP-сервера. Для доступа из Mac нужен `0.0.0.0` |
| `MT5_ADAPTER_PORT` | `8080` | Порт |
| `MT5_ALLOWED_ACCOUNT_MODE` | `demo` | `demo` / `real` — на какой тип счёта разрешены ордера |

## Запуск

```bash
pip3 install -r python/mt5_adapter/requirements.txt
python3 python/mt5_adapter/app.py
```

Либо `task run:mt5-adapter`.
