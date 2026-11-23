package telegram_v1

import (
	"errors"
	"testing"
)

const example = `ШОРТ

GOLD | ЗОЛОТО | XAUUSD

Зона входа: 4167.55

Цели:
→ TP1: 4163.38
→ TP2: 4159.21
→ TP3: 4155.05

Стоп: 4175.89

#SOFT`

func TestDecode(t *testing.T) {
	got, err := Decode(example)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := Signal{
		Side:   SideSell,
		Symbol: "GOLD",
		SL:     4175.89,
		TP1:    4163.38,
		TP2:    4159.21,
		TP3:    4155.05,
		Entry:  4167.55,
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDecodeLongWithoutOptionalTP(t *testing.T) {
	got, err := Decode("ЛОНГ\nEURUSD | EURUSD\nСтоп: 1.07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Side != SideBuy {
		t.Fatalf("side: got %s, want buy", got.Side)
	}
	if got.Symbol != "EURUSD" {
		t.Fatalf("symbol: got %s, want EURUSD", got.Symbol)
	}
	if got.SL != 1.07 {
		t.Fatalf("sl: got %v, want 1.07", got.SL)
	}
	if got.TP1 != 0 {
		t.Fatalf("tp1 must be optional, got %v", got.TP1)
	}
}

func TestDecodeRequired(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr error
	}{
		{name: "missing side", text: "GOLD | XAUUSD\nСтоп: 1", wantErr: ErrMissingSide},
		{name: "missing symbol", text: "ШОРТ\nСтоп: 1", wantErr: ErrMissingSymbol},
		{name: "missing sl", text: "ШОРТ\nGOLD | XAUUSD", wantErr: ErrMissingSL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decode(tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}
