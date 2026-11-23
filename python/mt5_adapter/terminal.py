from __future__ import annotations

import logging
import threading
from typing import Any

log = logging.getLogger("mt5-adapter")


# Биты маски `symbol_info().filling_mode`. В Python-пакете MetaTrader5 этих констант нет
# (они есть только в MQL5), поэтому значения заданы здесь по документации MQL5.
SYMBOL_FILLING_FOK = 1
SYMBOL_FILLING_IOC = 2


def pick_filling_mode(mt5: Any, symbol_filling_mode: int) -> int:
    """Выбирает режим исполнения, который поддерживает символ у брокера.

    `symbol_info().filling_mode` — битовая маска разрешённых режимов (FOK=1, IOC=2).
    Предпочитаем IOC, затем FOK; если ни один не разрешён — RETURN.
    """
    if symbol_filling_mode & SYMBOL_FILLING_IOC:
        return mt5.ORDER_FILLING_IOC
    if symbol_filling_mode & SYMBOL_FILLING_FOK:
        return mt5.ORDER_FILLING_FOK
    return mt5.ORDER_FILLING_RETURN


ACCOUNT_MODES = ("demo", "real")


def account_mode(mt5: Any, trade_mode: int) -> str:
    """Переводит `account_info().trade_mode` в demo / real / contest."""
    if trade_mode == mt5.ACCOUNT_TRADE_MODE_DEMO:
        return "demo"
    if trade_mode == mt5.ACCOUNT_TRADE_MODE_REAL:
        return "real"
    return "contest"


def order_comment(label: str) -> str:
    """Комментарий ордера в MT5: "trader" или "trader tp1". MT5 обрезает комментарий до 31 символа."""
    return f"trader {label}".strip()[:31]


class Terminal:
    """Связь с локальным MT5 через официальный Python API (Windows Named Pipe)."""

    def __init__(self, allowed_account_mode: str = "demo") -> None:
        if allowed_account_mode not in ACCOUNT_MODES:
            raise ValueError(f"allowed_account_mode must be one of {ACCOUNT_MODES}, got {allowed_account_mode!r}")

        self._mt5: Any = None
        self.live = False
        self.allowed_account_mode = allowed_account_mode
        # ThreadingHTTPServer обрабатывает запросы в разных потоках, а MT5 API
        # не рассчитан на параллельные вызовы: ордера отправляем строго по одному.
        self._send_lock = threading.Lock()

    def connect(self) -> None:
        try:
            import MetaTrader5 as mt5
        except ImportError:
            log.warning("MetaTrader5 is not installed, running in dry-run")
            return

        self._mt5 = mt5
        self.live = bool(mt5.initialize())
        if not self.live:
            log.warning("MetaTrader5.initialize() failed, running in dry-run")
            return

        account = self.account()
        if account is None:
            log.warning("MT5 account info is unavailable, orders will be rejected")
            return

        log.info(
            "mt5 account login=%s server=%s mode=%s (allowed: %s)",
            account["login"],
            account["server"],
            account["mode"],
            self.allowed_account_mode,
        )
        if account["mode"] != self.allowed_account_mode:
            log.warning("account mode %s is not allowed, orders will be rejected", account["mode"])

    def account(self) -> dict | None:
        """Текущий счёт терминала. Читается каждый раз: в терминале можно переключить счёт."""
        if not self.live or self._mt5 is None:
            return None

        info = self._mt5.account_info()
        if info is None:
            return None

        return {
            "login": info.login,
            "server": info.server,
            "mode": account_mode(self._mt5, info.trade_mode),
        }

    def shutdown(self) -> None:
        if self.live and self._mt5 is not None:
            self._mt5.shutdown()

    def send_order(self, side: str, symbol: str, volume: float, sl: float, tp: float, comment: str = "") -> dict:
        if not self.live or self._mt5 is None:
            log.info("dry-run %s %s volume=%s sl=%s tp=%s comment=%s", side, symbol, volume, sl, tp, comment)
            return {"ok": True, "mode": "dry-run"}

        with self._send_lock:
            return self._send_live(side, symbol, volume, sl, tp, comment)

    def _send_live(self, side: str, symbol: str, volume: float, sl: float, tp: float, comment: str) -> dict:
        account = self.account()
        if account is None:
            return {"ok": False, "error": "cannot verify MT5 account, order rejected"}
        if account["mode"] != self.allowed_account_mode:
            return {
                "ok": False,
                "error": (
                    f"account mode {account['mode']} (login {account['login']}) is not allowed: "
                    f"MT5_ALLOWED_ACCOUNT_MODE={self.allowed_account_mode}"
                ),
            }

        mt5 = self._mt5
        if not mt5.symbol_select(symbol, True):
            return {"ok": False, "error": f"unknown symbol {symbol}"}

        info = mt5.symbol_info(symbol)
        if info is None:
            return {"ok": False, "error": f"no symbol info for {symbol}"}

        tick = mt5.symbol_info_tick(symbol)
        if tick is None:
            return {"ok": False, "error": f"no tick for {symbol}"}

        order_type = mt5.ORDER_TYPE_BUY if side == "buy" else mt5.ORDER_TYPE_SELL
        price = tick.ask if side == "buy" else tick.bid
        request = {
            "action": mt5.TRADE_ACTION_DEAL,
            "symbol": symbol,
            "volume": volume,
            "type": order_type,
            "price": price,
            "deviation": 20,
            "magic": 20261005,
            "comment": order_comment(comment),
            "type_time": mt5.ORDER_TIME_GTC,
            "type_filling": pick_filling_mode(mt5, info.filling_mode),
        }
        if sl:
            request["sl"] = sl
        if tp:
            request["tp"] = tp

        result = mt5.order_send(request)
        if result is None:
            return {"ok": False, "error": str(mt5.last_error())}

        ok = result.retcode == mt5.TRADE_RETCODE_DONE
        return {
            "ok": ok,
            "mode": "live",
            "retcode": result.retcode,
            "comment": result.comment,
            "order": result.order,
            "deal": result.deal,
        }
