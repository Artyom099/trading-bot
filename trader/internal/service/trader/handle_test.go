package trader

import (
	"context"
	"errors"
	"testing"

	"github.com/Artyom099/trading-bot/trader/internal/model"
)

const (
	testChatID = int64(872904031)
	testVolume = 0.01
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
	svc := NewService(adapter, testChatID, testVolume)

	err := svc.HandleMessage(context.Background(), testChatID+1, validText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for other chat, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageSkipsInvalidText(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolume)

	err := svc.HandleMessage(context.Background(), testChatID, "hello")
	if err != nil {
		t.Fatalf("invalid text must be skipped, got error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called for invalid text, got %d calls", len(adapter.signals))
	}
}

func TestHandleMessageOpensOrder(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc := NewService(adapter, testChatID, testVolume)

	err := svc.HandleMessage(context.Background(), testChatID, validText)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := model.Signal{
		Side:   model.SideSell,
		Symbol: "XAUUSD",
		Volume: testVolume,
		Entry:  4167.55,
		SL:     4175.89,
		TP:     4163.38,
		TP2:    4159.21,
		TP3:    4155.05,
	}
	if len(adapter.signals) != 1 {
		t.Fatalf("OpenOrder calls: got %d, want 1", len(adapter.signals))
	}
	if adapter.signals[0] != want {
		t.Fatalf("got %+v, want %+v", adapter.signals[0], want)
	}
}

func TestHandleMessageOpenOrderError(t *testing.T) {
	openErr := errors.New("mt5 adapter down")
	adapter := &mt5AdapterStub{err: openErr}
	svc := NewService(adapter, testChatID, testVolume)

	err := svc.HandleMessage(context.Background(), testChatID, validText)
	if !errors.Is(err, openErr) {
		t.Fatalf("got %v, want %v", err, openErr)
	}
	if len(adapter.signals) != 1 {
		t.Fatalf("OpenOrder calls: got %d, want 1", len(adapter.signals))
	}
}
