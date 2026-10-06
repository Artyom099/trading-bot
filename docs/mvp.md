# Сервис для копи-трейдинга

## Описание

Один Go-сервис `trader` слушает один Telegram-чат. Я пишу туда сообщение — сервис сразу открывает сделку в реальном времени.

Go отвечает за Telegram и бизнес-логику. MT5 только исполняет ордер: сервис дергает Python-адаптер по HTTP, адаптер вызывает `order_send()` в запущенном терминале.

## Поток

```text
Я пишу в Telegram-чат
    │
    ▼
trader (Go)
    │ парсит сигнал, решает что открыть
    │ HTTP
    ▼
Python MT5 Adapter
    │ order_send()
    ▼
MT5 Terminal
    │
    ▼
Broker
```

## Формат сообщений из телеграма

Контракт: [`shared/api/telegram/v1/signal.yaml`](../shared/api/telegram/v1/signal.yaml).

Одно сообщение — одна сделка:

```text
ШОРТ

GOLD | ЗОЛОТО | XAUUSD

Зона входа: 4167.55

Цели:
→ TP1: 4163.38
→ TP2: 4159.21
→ TP3: 4155.05

Стоп: 4175.89

#SOFT
```

- `ШОРТ` = `SELL`, ЛОНГ = BUY — в какую сторону открывается сделка, на покупку или на продажу
- GOLD | ЗОЛОТО | XAUUSD - название актива
- Зона входа: 4167.55 - пока не трогаем
- объём в лотах - задается из переменных окружения, по умлочанию 0,01
- Стоп: 4175.89 = `SL` - он обязателен
- TP1, TP2, TP3 - `TP` - они необязательны
- #SOFT - не используем

Чужие чаты сервис игнорирует.

## Запуск на Mac (dry-run)

Без Windows адаптер стартует, но ордера в рынок не уходят (`MetaTrader5 is not installed, running in dry-run`).

1. Заполнить `deploy/compose/trader/.env`: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_CHAT_ID`, `TRADE_VOLUME`.
2. Для канала бот должен быть админом, чтобы получать `channel_post`. Для личного чата с ботом достаточно `/start`.
3. Запуск:

```bash
task run:mt5-adapter
task run:trader
```

## Windows в виртуалке

На Mac нет официального `MetaTrader5` Python API. Нормальный путь без отдельного Windows-ПК: виртуалка с Windows 11. Внутри неё — терминал MT5 и Python-адаптер. `trader` остаётся на Mac и ходит в адаптер по HTTP.

```text
Mac                              Windows VM
─────────────────────            ────────────────────────────────
Telegram → trader (Go)
                 │
                 │  HTTP  MT5_ADAPTER_URL=http://<IP-VM>:8080
                 ▼
                 Python MT5 Adapter  →  terminal64.exe  →  брокер
```

Адаптер и терминал должны быть на одной Windows-машине (Named Pipe). `trader` может быть где угодно, лишь бы достучался до `:8080`.

### Гипервизор

| Программа | Зачем |
|---|---|
| [UTM](https://mac.getutm.app/) | бесплатно, нормально на Apple Silicon |
| Parallels Desktop | проще сеть и шаринг папок, платно |
| VMware Fusion | тоже ок, если уже стоит |

На M1/M2/M3 нужна Windows 11 ARM (ISO с сайта Microsoft), не x86.

Ресурсы примерно: 4+ CPU, 8 GB RAM, 64 GB диск. Без GUI терминала MT5 не залогиниться в брокер.

Сеть: Bridged (виртуалка в той же LAN, что Mac) — тогда у VM свой IP, и `trader` ходит на `http://192.168.x.x:8080`. Либо NAT + проброс порта `Mac:8080 → VM:8080`, тогда в `.env` можно оставить `http://127.0.0.1:8080`.

### Внутри Windows

1. Поставить [Python](https://www.python.org/downloads/windows/) 3.10+ (галка Add python.exe to PATH).
2. Склонировать репозиторий (или скопировать папку `python/mt5_adapter`).
3. Установить пакет:

```bat
pip install -r python\mt5_adapter\requirements.txt
```

4. Скачать MT5 с сайта брокера, залогиниться в счёт, включить Algo Trading (автоторговля). Терминал не закрывать.
5. Слушать не только localhost, иначе Mac не достучится:

```bat
set MT5_ADAPTER_HOST=0.0.0.0
python python\mt5_adapter\app.py
```

В логе должно быть `live=True`. Если `dry-run` — пакет не встал или `initialize()` не нашёл терминал.

6. Windows Firewall: входящий TCP 8080. Иначе с Mac будет timeout.
7. Узнать IP виртуалки: `ipconfig` → IPv4 (например `192.168.1.42`).

Проверка с Mac:

```bash
curl http://192.168.1.42:8080/health
```

Ожидается `{"status": "ok", "live": true}`.

### На Mac

В `deploy/compose/trader/.env`:

```env
MT5_ADAPTER_URL=http://192.168.1.42:8080
```

Дальше как обычно: `task run:trader`. Адаптер на Mac не запускать — он должен жить в VM рядом с терминалом.

### Если не коннектится

- Адаптер слушает `127.0.0.1` — с Mac его не видно. Нужен `MT5_ADAPTER_HOST=0.0.0.0`.
- Firewall режет 8080.
- IP виртуалки сменился после DHCP — снова `ipconfig` и поправить `.env`.
- Терминал не запущен или автоторговля выключена — будет `live=false` или `502` на ордере.
