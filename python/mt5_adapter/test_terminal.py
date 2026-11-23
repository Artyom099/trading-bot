from __future__ import annotations

import sys
import threading
import time
import unittest
from types import SimpleNamespace
from unittest import mock

from terminal import Terminal, order_comment, pick_filling_mode


def fake_mt5(
    filling_mode: int = 2,
    symbol_exists: bool = True,
    trade_mode: int = 0,
    account_exists: bool = True,
    symbol_selectable: bool = True,
    tick_exists: bool = True,
    send_result: object = "ok",
    initialize_ok: bool = True,
) -> SimpleNamespace:
    # Как и в настоящем пакете MetaTrader5, здесь нет SYMBOL_FILLING_FOK/IOC: они только в MQL5.
    sent: list[dict] = []

    def order_send(request: dict) -> SimpleNamespace | None:
        sent.append(request)
        if send_result == "ok":
            return SimpleNamespace(retcode=10009, comment="done", order=1, deal=2)
        return send_result  # type: ignore[return-value]

    return SimpleNamespace(
        ORDER_FILLING_FOK=0,
        ORDER_FILLING_IOC=1,
        ORDER_FILLING_RETURN=2,
        ORDER_TYPE_BUY=0,
        ORDER_TYPE_SELL=1,
        TRADE_ACTION_DEAL=1,
        ORDER_TIME_GTC=0,
        TRADE_RETCODE_DONE=10009,
        ACCOUNT_TRADE_MODE_DEMO=0,
        ACCOUNT_TRADE_MODE_CONTEST=1,
        ACCOUNT_TRADE_MODE_REAL=2,
        account_info=lambda: (
            SimpleNamespace(login=12345, server="Broker-Demo", trade_mode=trade_mode) if account_exists else None
        ),
        symbol_select=lambda symbol, enable: symbol_selectable,
        symbol_info=lambda symbol: SimpleNamespace(filling_mode=filling_mode) if symbol_exists else None,
        symbol_info_tick=lambda symbol: SimpleNamespace(ask=101.0, bid=100.0) if tick_exists else None,
        initialize=lambda: initialize_ok,
        shutdown=lambda: None,
        order_send=order_send,
        last_error=lambda: (1, "error"),
        sent=sent,
    )


class PickFillingModeTest(unittest.TestCase):
    def test_modes(self) -> None:
        mt5 = fake_mt5(0)
        cases = {
            0: mt5.ORDER_FILLING_RETURN,
            1: mt5.ORDER_FILLING_FOK,
            2: mt5.ORDER_FILLING_IOC,
            3: mt5.ORDER_FILLING_IOC,  # FOK|IOC -> предпочитаем IOC
        }
        for mask, want in cases.items():
            with self.subTest(mask=mask):
                self.assertEqual(pick_filling_mode(mt5, mask), want)


class SendOrderFillingTest(unittest.TestCase):
    def _terminal(self, mt5: SimpleNamespace, allowed: str = "demo") -> Terminal:
        terminal = Terminal(allowed)
        terminal._mt5 = mt5
        terminal.live = True
        return terminal

    def test_uses_fok_when_ioc_unsupported(self) -> None:
        mt5 = fake_mt5(filling_mode=1)
        result = self._terminal(mt5).send_order("buy", "XAUUSD", 0.01, 90.0, 110.0)
        self.assertTrue(result["ok"])
        self.assertEqual(mt5.sent[0]["type_filling"], mt5.ORDER_FILLING_FOK)

    def test_uses_return_when_nothing_flagged(self) -> None:
        mt5 = fake_mt5(filling_mode=0)
        self._terminal(mt5).send_order("sell", "XAUUSD", 0.01, 110.0, 90.0)
        self.assertEqual(mt5.sent[0]["type_filling"], mt5.ORDER_FILLING_RETURN)

    def test_missing_symbol_info(self) -> None:
        mt5 = fake_mt5(filling_mode=2, symbol_exists=False)
        result = self._terminal(mt5).send_order("buy", "XAUUSD", 0.01, 0, 0)
        self.assertFalse(result["ok"])
        self.assertEqual(mt5.sent, [])


class SendOrderErrorsTest(unittest.TestCase):
    def _send(self, mt5: SimpleNamespace, side: str = "buy", sl: float = 90.0, tp: float = 110.0) -> dict:
        terminal = Terminal("demo")
        terminal._mt5 = mt5
        terminal.live = True
        return terminal.send_order(side, "XAUUSD", 0.01, sl, tp)

    def test_unknown_symbol(self) -> None:
        mt5 = fake_mt5(symbol_selectable=False)
        result = self._send(mt5)
        self.assertFalse(result["ok"])
        self.assertIn("unknown symbol", result["error"])
        self.assertEqual(mt5.sent, [])

    def test_no_tick(self) -> None:
        mt5 = fake_mt5(tick_exists=False)
        result = self._send(mt5)
        self.assertFalse(result["ok"])
        self.assertIn("no tick", result["error"])
        self.assertEqual(mt5.sent, [])

    def test_order_send_returns_none(self) -> None:
        mt5 = fake_mt5(send_result=None)
        result = self._send(mt5)
        self.assertFalse(result["ok"])
        self.assertIn("error", result["error"])  # str(last_error())

    def test_rejected_by_broker(self) -> None:
        rejected = SimpleNamespace(retcode=10016, comment="Invalid stops", order=0, deal=0)
        result = self._send(fake_mt5(send_result=rejected))
        self.assertFalse(result["ok"])
        self.assertEqual(result["retcode"], 10016)
        self.assertEqual(result["comment"], "Invalid stops")

    def test_successful_order_result(self) -> None:
        result = self._send(fake_mt5())
        self.assertEqual(
            result,
            {"ok": True, "mode": "live", "retcode": 10009, "comment": "done", "order": 1, "deal": 2},
        )


class SendOrderRequestTest(unittest.TestCase):
    def _request(self, side: str, sl: float, tp: float) -> dict:
        mt5 = fake_mt5()
        terminal = Terminal("demo")
        terminal._mt5 = mt5
        terminal.live = True
        terminal.send_order(side, "XAUUSD", 0.01, sl, tp)
        return mt5.sent[0]

    def test_buy_uses_ask_and_sets_sl_tp(self) -> None:
        req = self._request("buy", 90.0, 110.0)
        self.assertEqual(req["type"], 0)
        self.assertEqual(req["price"], 101.0)
        self.assertEqual((req["sl"], req["tp"]), (90.0, 110.0))
        self.assertEqual((req["symbol"], req["volume"]), ("XAUUSD", 0.01))

    def test_sell_uses_bid(self) -> None:
        req = self._request("sell", 110.0, 90.0)
        self.assertEqual(req["type"], 1)
        self.assertEqual(req["price"], 100.0)

    def test_comment_marks_target(self) -> None:
        mt5 = fake_mt5()
        terminal = Terminal("demo")
        terminal._mt5 = mt5
        terminal.live = True
        terminal.send_order("buy", "XAUUSD", 0.02, 90.0, 110.0, "tp2")
        terminal.send_order("buy", "XAUUSD", 0.02, 90.0, 110.0)
        self.assertEqual([r["comment"] for r in mt5.sent], ["trader tp2", "trader"])

    def test_comment_is_truncated_to_31_chars(self) -> None:
        self.assertEqual(len(order_comment("x" * 100)), 31)

    def test_zero_sl_tp_are_omitted(self) -> None:
        req = self._request("buy", 0, 0)
        self.assertNotIn("sl", req)
        self.assertNotIn("tp", req)


class SendOrderConcurrencyTest(unittest.TestCase):
    def test_orders_are_sent_one_at_a_time(self) -> None:
        mt5 = fake_mt5()
        state = {"active": 0, "max_active": 0}
        guard = threading.Lock()

        def slow_order_send(request: dict) -> SimpleNamespace:
            with guard:
                state["active"] += 1
                state["max_active"] = max(state["max_active"], state["active"])
            time.sleep(0.02)
            with guard:
                state["active"] -= 1
            return SimpleNamespace(retcode=10009, comment="done", order=1, deal=2)

        mt5.order_send = slow_order_send
        terminal = Terminal("demo")
        terminal._mt5 = mt5
        terminal.live = True

        results: list[dict] = []
        threads = [
            threading.Thread(target=lambda: results.append(terminal.send_order("buy", "XAUUSD", 0.01, 90.0, 110.0)))
            for _ in range(5)
        ]
        for th in threads:
            th.start()
        for th in threads:
            th.join()

        self.assertEqual(len(results), 5)
        self.assertTrue(all(r["ok"] for r in results))
        self.assertEqual(state["max_active"], 1)


class ConnectTest(unittest.TestCase):
    def test_package_not_installed_means_dry_run(self) -> None:
        terminal = Terminal()
        with mock.patch.dict(sys.modules, {"MetaTrader5": None}):  # import -> ImportError
            terminal.connect()
        self.assertFalse(terminal.live)
        self.assertIsNone(terminal.account())

    def test_initialize_failure_means_dry_run(self) -> None:
        terminal = Terminal()
        with mock.patch.dict(sys.modules, {"MetaTrader5": fake_mt5(initialize_ok=False)}):
            terminal.connect()
        self.assertFalse(terminal.live)
        self.assertEqual(terminal.send_order("buy", "XAUUSD", 0.01, 0, 0), {"ok": True, "mode": "dry-run"})

    def test_successful_connect_reports_account(self) -> None:
        terminal = Terminal()
        with mock.patch.dict(sys.modules, {"MetaTrader5": fake_mt5(trade_mode=2)}):
            terminal.connect()
        self.assertTrue(terminal.live)
        self.assertEqual(terminal.account(), {"login": 12345, "server": "Broker-Demo", "mode": "real"})


class AccountGuardTest(unittest.TestCase):
    def _send(self, mt5: SimpleNamespace, allowed: str) -> dict:
        terminal = Terminal(allowed)
        terminal._mt5 = mt5
        terminal.live = True
        return terminal.send_order("buy", "XAUUSD", 0.01, 90.0, 110.0)

    def test_real_account_rejected_by_default(self) -> None:
        mt5 = fake_mt5(trade_mode=2)
        result = self._send(mt5, "demo")
        self.assertFalse(result["ok"])
        self.assertIn("real", result["error"])
        self.assertEqual(mt5.sent, [])

    def test_demo_account_rejected_when_real_required(self) -> None:
        mt5 = fake_mt5(trade_mode=0)
        self.assertFalse(self._send(mt5, "real")["ok"])
        self.assertEqual(mt5.sent, [])

    def test_matching_mode_allowed(self) -> None:
        for trade_mode, allowed in ((0, "demo"), (2, "real")):
            with self.subTest(allowed=allowed):
                mt5 = fake_mt5(trade_mode=trade_mode)
                self.assertTrue(self._send(mt5, allowed)["ok"])
                self.assertEqual(len(mt5.sent), 1)

    def test_contest_account_rejected(self) -> None:
        mt5 = fake_mt5(trade_mode=1)
        self.assertFalse(self._send(mt5, "demo")["ok"])

    def test_unverifiable_account_rejected(self) -> None:
        mt5 = fake_mt5(account_exists=False)
        self.assertFalse(self._send(mt5, "demo")["ok"])
        self.assertEqual(mt5.sent, [])

    def test_dry_run_ignores_account(self) -> None:
        result = Terminal("demo").send_order("buy", "XAUUSD", 0.01, 0, 0)
        self.assertEqual(result, {"ok": True, "mode": "dry-run"})

    def test_invalid_allowed_mode(self) -> None:
        with self.assertRaises(ValueError):
            Terminal("any")


if __name__ == "__main__":
    unittest.main()
