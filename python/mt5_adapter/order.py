from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class Order:
    side: str
    symbol: str
    volume: float
    sl: float
    tp: float


def parse_order(body: dict) -> Order:
    side = body["side"]
    symbol = body["symbol"]
    volume = float(body["volume"])
    sl = float(body.get("sl") or 0)
    tp = float(body.get("tp") or 0)

    if side not in ("buy", "sell") or volume <= 0:
        raise ValueError("side must be buy/sell and volume > 0")

    return Order(side=side, symbol=symbol, volume=volume, sl=sl, tp=tp)
