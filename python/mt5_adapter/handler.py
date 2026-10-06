from __future__ import annotations

import json
import logging
from http.server import BaseHTTPRequestHandler
from typing import Any

from order import parse_order
from terminal import Terminal

log = logging.getLogger("mt5-adapter")


def new_handler(terminal: Terminal) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def log_message(self, fmt: str, *args: Any) -> None:
            log.info("%s - %s", self.address_string(), fmt % args)

        def do_GET(self) -> None:
            if self.path != "/health":
                self._json(404, {"error": "not found"})
                return
            self._json(200, {"status": "ok", "live": terminal.live})

        def do_POST(self) -> None:
            if self.path != "/v1/orders":
                self._json(404, {"error": "not found"})
                return

            length = int(self.headers.get("Content-Length", "0"))
            try:
                body = json.loads(self.rfile.read(length))
                order = parse_order(body)
            except (KeyError, TypeError, ValueError, json.JSONDecodeError) as err:
                self._json(400, {"error": f"invalid body: {err}"})
                return

            result = terminal.send_order(order.side, order.symbol, order.volume, order.sl, order.tp)
            status = 200 if result.get("ok") else 502
            self._json(status, result)

        def _json(self, status: int, payload: dict) -> None:
            body = json.dumps(payload).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return Handler
