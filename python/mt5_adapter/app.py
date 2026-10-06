#!/usr/bin/env python3
"""HTTP adapter: trader -> MetaTrader 5. Live orders need Windows + a running MT5 terminal."""

from __future__ import annotations

import logging
import os
from http.server import ThreadingHTTPServer

from handler import new_handler
from terminal import Terminal

HOST = os.getenv("MT5_ADAPTER_HOST", "127.0.0.1")
PORT = int(os.getenv("MT5_ADAPTER_PORT", "8080"))

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("mt5-adapter")


def main() -> None:
    terminal = Terminal()
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
