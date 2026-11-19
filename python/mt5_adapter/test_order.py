from __future__ import annotations

import unittest

from order import Order, parse_order


class ParseOrderTest(unittest.TestCase):
    def test_valid_sell(self) -> None:
        got = parse_order(
            {
                "side": "sell",
                "symbol": "XAUUSD",
                "volume": 0.01,
                "sl": 4175.89,
                "tp": 4163.38,
            }
        )
        self.assertEqual(
            got,
            Order(side="sell", symbol="XAUUSD", volume=0.01, sl=4175.89, tp=4163.38),
        )

    def test_optional_sl_tp_default_zero(self) -> None:
        got = parse_order({"side": "buy", "symbol": "EURUSD", "volume": 0.1})
        self.assertEqual(got.sl, 0.0)
        self.assertEqual(got.tp, 0.0)

    def test_null_sl_tp_default_zero(self) -> None:
        got = parse_order(
            {"side": "buy", "symbol": "EURUSD", "volume": 0.1, "sl": None, "tp": None}
        )
        self.assertEqual(got.sl, 0.0)
        self.assertEqual(got.tp, 0.0)

    def test_comment_is_optional(self) -> None:
        body = {"side": "buy", "symbol": "XAUUSD", "volume": 0.01}
        self.assertEqual(parse_order(body).comment, "")
        self.assertEqual(parse_order({**body, "comment": "tp2"}).comment, "tp2")

    def test_invalid_side(self) -> None:
        with self.assertRaises(ValueError):
            parse_order({"side": "long", "symbol": "XAUUSD", "volume": 0.01})

    def test_non_positive_volume(self) -> None:
        for volume in (0, -0.01):
            with self.subTest(volume=volume):
                with self.assertRaises(ValueError):
                    parse_order({"side": "buy", "symbol": "XAUUSD", "volume": volume})

    def test_missing_required(self) -> None:
        body = {"side": "buy", "symbol": "XAUUSD", "volume": 0.01}
        for key in ("side", "symbol", "volume"):
            with self.subTest(key=key):
                incomplete = dict(body)
                del incomplete[key]
                with self.assertRaises(KeyError):
                    parse_order(incomplete)


if __name__ == "__main__":
    unittest.main()
