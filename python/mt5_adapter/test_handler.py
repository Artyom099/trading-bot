from __future__ import annotations

import json
import logging
import threading
import unittest
import urllib.error
import urllib.request
from http.server import ThreadingHTTPServer
from typing import Any

from handler import new_handler


class StubTerminal:
    """Подмена Terminal: возвращает заданный результат или бросает исключение."""

    live = True
    allowed_account_mode = "demo"

    def __init__(self, result: dict | None = None, error: Exception | None = None) -> None:
        self.result = result or {"ok": True, "mode": "live"}
        self.error = error

    def account(self) -> dict | None:
        if self.error:
            raise self.error
        return {"login": 1, "server": "Broker-Demo", "mode": "demo"}

    def send_order(self, *args: Any) -> dict:
        if self.error:
            raise self.error
        return self.result


class HandlerTest(unittest.TestCase):
    def setUp(self) -> None:
        # Ожидаемые исключения пишутся в лог адаптера, в выводе тестов они не нужны.
        logging.getLogger("mt5-adapter").setLevel(logging.CRITICAL)

    def _serve(self, terminal: StubTerminal) -> str:
        server = ThreadingHTTPServer(("127.0.0.1", 0), new_handler(terminal))  # type: ignore[arg-type]
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(thread.join, 5)
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)

        return f"http://127.0.0.1:{server.server_address[1]}"

    def _request(self, url: str, body: dict | str | None = None) -> tuple[int, dict]:
        data = None
        if body is not None:
            data = (body if isinstance(body, str) else json.dumps(body)).encode()

        request = urllib.request.Request(url, data=data, headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(request, timeout=5) as response:
                return response.status, json.loads(response.read())
        except urllib.error.HTTPError as err:
            return err.code, json.loads(err.read())

    ORDER = {"side": "buy", "symbol": "GOLD", "volume": 0.01, "sl": 90.0, "tp": 110.0}

    def test_order_ok(self) -> None:
        base = self._serve(StubTerminal())
        status, body = self._request(base + "/v1/orders", self.ORDER)
        self.assertEqual((status, body["ok"]), (200, True))

    def test_order_rejected_by_terminal(self) -> None:
        base = self._serve(StubTerminal(result={"ok": False, "error": "unknown symbol GOLD"}))
        status, body = self._request(base + "/v1/orders", self.ORDER)
        self.assertEqual(status, 502)
        self.assertEqual(body["error"], "unknown symbol GOLD")

    def test_order_exception_becomes_502(self) -> None:
        base = self._serve(StubTerminal(error=AttributeError("no attribute SYMBOL_FILLING_IOC")))
        status, body = self._request(base + "/v1/orders", self.ORDER)
        self.assertEqual(status, 502)
        self.assertFalse(body["ok"])
        self.assertIn("SYMBOL_FILLING_IOC", body["error"])

    def test_health_exception_becomes_500(self) -> None:
        base = self._serve(StubTerminal(error=RuntimeError("terminal gone")))
        status, body = self._request(base + "/health")
        self.assertEqual(status, 500)
        self.assertIn("terminal gone", body["error"])

    def test_invalid_body(self) -> None:
        base = self._serve(StubTerminal())
        status, _ = self._request(base + "/v1/orders", {"side": "hold"})
        self.assertEqual(status, 400)

    def test_unknown_path(self) -> None:
        base = self._serve(StubTerminal())
        status, _ = self._request(base + "/nope")
        self.assertEqual(status, 404)


if __name__ == "__main__":
    unittest.main()
