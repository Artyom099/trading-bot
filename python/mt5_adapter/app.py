#!/usr/bin/env python3
"""HTTP adapter: trader -> MetaTrader 5. Live orders need Windows + a running MT5 terminal."""

from __future__ import annotations

import logging
import os
from http.server import ThreadingHTTPServer
from pathlib import Path

from dotenv import load_dotenv

from handler import new_handler
from terminal import Terminal

# .env лежит рядом с app.py (не зависит от текущей директории). Уже заданные
# переменные окружения имеют приоритет над файлом. Нет файла — не ошибка.
load_dotenv(Path(__file__).resolve().parent / ".env")

HOST = os.getenv("MT5_ADAPTER_HOST", "127.0.0.1")
PORT = int(os.getenv("MT5_ADAPTER_PORT", "8080"))
ALLOWED_ACCOUNT_MODE = os.getenv("MT5_ALLOWED_ACCOUNT_MODE", "demo").strip().lower()

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("mt5-adapter")


def main() -> None:
    terminal = Terminal(ALLOWED_ACCOUNT_MODE)
    terminal.connect()

    server = ThreadingHTTPServer((HOST, PORT), new_handler(terminal))
    log.info("mt5 adapter listening on http://%s:%s live=%s", HOST, PORT, terminal.live)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
        terminal.shutdown()


if __name__ == "__main__":
    main()
