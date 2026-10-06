from __future__ import annotations

import logging
from typing import Any

log = logging.getLogger("mt5-adapter")


class Terminal:
    """Связь с локальным MT5 через официальный Python API (Windows Named Pipe)."""

    def __init__(self) -> None:
        self._mt5: Any = None
        self.live = False

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

    def shutdown(self) -> None:
        if self.live and self._mt5 is not None:
            self._mt5.shutdown()

    def send_order(self, side: str, symbol: str, volume: float, sl: float, tp: float) -> dict:
        if not self.live or self._mt5 is None:
            log.info("dry-run %s %s volume=%s sl=%s tp=%s", side, symbol, volume, sl, tp)
            return {"ok": True, "mode": "dry-run"}

        mt5 = self._mt5
        if not mt5.symbol_select(symbol, True):
            return {"ok": False, "error": f"unknown symbol {symbol}"}

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
            "comment": "trader",
            "type_time": mt5.ORDER_TIME_GTC,
            "type_filling": mt5.ORDER_FILLING_IOC,
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
