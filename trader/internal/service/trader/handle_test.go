package trader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Artyom099/trading-bot/trader/internal/killswitch"
	"github.com/Artyom099/trading-bot/trader/internal/model"
)

var testVolumes = model.TPVolumes{TP1: 0.01, TP2: 0.02, TP3: 0.03}

const (
	testChatID = int64(872904031)
	testMaxAge = time.Minute
	validText  = `ШОРТ

GOLD | ЗОЛОТО | XAUUSD

Зона входа: 4167.55

Цели:
→ TP1: 4163.38
→ TP2: 4159.21
→ TP3: 4155.05

Стоп: 4175.89

#SOFT`
)

func newKillSwitch(t *testing.T, cfg killswitch.Config) *killswitch.Switch {
	t.Helper()

	ks, err := killswitch.New(cfg)
	if err != nil {
		t.Fatalf("killswitch.New: %v", err)
	}

	return ks
}

type mt5AdapterStub struct {
	signals []model.Signal
	err     error
}

func (m *mt5AdapterStub) OpenOrder(_ context.Context, signal model.Signal) error {
	m.signals = append(m.signals, signal)
	return m.err
}

func TestHandleMessageIgnoresOtherChat(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID+1, validText, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for other chat, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageSkipsInvalidText(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID, "hello", time.Now())
	if err != nil {
		t.Fatalf("invalid text must be skipped, got error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for invalid text, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageOpensTradePerTarget(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	base := model.Signal{Side: model.SideSell, Symbol: "GOLD", Entry: 4167.55, SL: 4175.89}
	want := []model.Signal{
		{Side: base.Side, Symbol: base.Symbol, Entry: base.Entry, SL: base.SL, TP: 4163.38, Volume: 0.01, Comment: "tp1"},
		{Side: base.Side, Symbol: base.Symbol, Entry: base.Entry, SL: base.SL, TP: 4159.21, Volume: 0.02, Comment: "tp2"},
		{Side: base.Side, Symbol: base.Symbol, Entry: base.Entry, SL: base.SL, TP: 4155.05, Volume: 0.03, Comment: "tp3"},
	}
	if len(adapter.signals) != len(want) {
		t.Fatalf("OpenOrder calls: got %d, want %d", len(adapter.signals), len(want))
	}
	for i := range want {
		if adapter.signals[i] != want[i] {
			t.Fatalf("trade %d: got %+v, want %+v", i, adapter.signals[i], want[i])
		}
	}
}

func TestHandleMessageOnlyPresentTargets(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	text := "ЛОНГ\nGOLD | XAUUSD\nTP1: 4170.00\nTP3: 4180.00\nСтоп: 4150.00"
	if err := svc.HandleMessage(context.Background(), testChatID, text, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(adapter.signals) != 2 {
		t.Fatalf("OpenOrder calls: got %d, want 2", len(adapter.signals))
	}
	if got := adapter.signals[0]; got.Comment != "tp1" || got.TP != 4170 || got.Volume != 0.01 {
		t.Fatalf("first trade: %+v", got)
	}
	if got := adapter.signals[1]; got.Comment != "tp3" || got.TP != 4180 || got.Volume != 0.03 {
		t.Fatalf("second trade: %+v", got)
	}
}

func TestHandleMessageSingleTargetOpensOneTrade(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	text := "ШОРТ\n\nGOLD | ЗОЛОТО | XAUUSD\n\nЗона входа: 3620.16\n\nЦель:\n→ TP: 3612.01\n\nСтоп: 3625.59\n\n#SOFT"
	if err := svc.HandleMessage(context.Background(), testChatID, text, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(adapter.signals) != 1 {
		t.Fatalf("OpenOrder calls: got %d, want 1", len(adapter.signals))
	}
	want := model.Signal{
		Side: model.SideSell, Symbol: "GOLD", Entry: 3620.16, SL: 3625.59, TP: 3612.01, Volume: testVolumes.TP1, Comment: "tp1",
	}
	if adapter.signals[0] != want {
		t.Fatalf("trade: got %+v, want %+v", adapter.signals[0], want)
	}
}

func TestNoTradesMessage(t *testing.T) {
	tests := []struct {
		name   string
		signal model.Signal
		want   string
	}{
		{name: "no targets", signal: model.Signal{SL: 1}, want: msgNoTargets},
		{name: "targets exist, volume zero", signal: model.Signal{SL: 1, TP: 2}, want: msgZeroVolume},
		{name: "only tp3", signal: model.Signal{SL: 1, TP3: 2}, want: msgZeroVolume},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noTradesMessage(tt.signal); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleMessageNoTargetsOpensNothing(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	text := "ШОРТ\nGOLD | XAUUSD\nСтоп: 4175.89"
	if err := svc.HandleMessage(context.Background(), testChatID, text, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for signal without targets, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageZeroVolumeSkipsTarget(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, model.TPVolumes{TP1: 0.01, TP2: 0, TP3: 0.03}, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	if err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(adapter.signals) != 2 {
		t.Fatalf("OpenOrder calls: got %d, want 2", len(adapter.signals))
	}
	if adapter.signals[0].Comment != "tp1" || adapter.signals[1].Comment != "tp3" {
		t.Fatalf("got %q and %q, want tp1 and tp3", adapter.signals[0].Comment, adapter.signals[1].Comment)
	}
}

func TestHandleMessageAllVolumesZeroOpensNothing(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, model.TPVolumes{}, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	if err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageOpenOrderErrorStillTriesOtherTrades(t *testing.T) {
	openErr := errors.New("mt5 adapter down")
	adapter := &mt5AdapterStub{err: openErr}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now())
	if !errors.Is(err, openErr) {
		t.Fatalf("got %v, want %v", err, openErr)
	}
	if len(adapter.signals) != 3 {
		t.Fatalf("a failed trade must not cancel the others: OpenOrder calls: got %d, want 3", len(adapter.signals))
	}
}

func TestHandleMessageSkipsStaleMessage(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now().Add(-2*testMaxAge))
	if err != nil {
		t.Fatalf("stale message must be skipped, got error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for stale message, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageAgeFilterDisabled(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolumes, 0, newKillSwitch(t, killswitch.Config{}), nil)

	err := svc.HandleMessage(context.Background(), testChatID, validText, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 3 {
		t.Fatalf("OpenOrder calls: got %d, want 3", len(adapter.signals))
	}
}
