package trader

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Artyom099/trading-bot/trader/internal/killswitch"
)

func newTestService(t *testing.T, adapter *mt5AdapterStub, cfg killswitch.Config) (*service, *killswitch.Switch) {
	t.Helper()

	ks := newKillSwitch(t, cfg)

	return NewService(adapter, testChatID, testVolumes, testMaxAge, ks, &notifierStub{}), ks
}

type notifierStub struct {
	messages []string
	err      error
}

func (n *notifierStub) Notify(_ context.Context, text string) error {
	n.messages = append(n.messages, text)
	return n.err
}

func sentMessages(svc *service) []string {
	return svc.notifier.(*notifierStub).messages
}

func sendSignal(t *testing.T, svc *service) error {
	t.Helper()

	return svc.HandleMessage(context.Background(), testChatID, validText, time.Now())
}

func command(svc *service, text string) (string, bool) {
	return svc.HandleCommand(context.Background(), testChatID, text, time.Now())
}

func TestStoppedTradingBlocksSignals(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc, ks := newTestService(t, adapter, killswitch.Config{})
	if err := ks.Stop(killswitch.ReasonManual); err != nil {
		t.Fatal(err)
	}

	if err := sendSignal(t, svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(adapter.signals) != 0 {
		t.Fatalf("OpenOrder must not be called while stopped, got %d calls", len(adapter.signals))
	}
}

func TestStopCommandThenStartCommand(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc, _ := newTestService(t, adapter, killswitch.Config{})

	reply, handled := command(svc, "/stop")
	if !handled || !strings.Contains(reply, "остановлена") {
		t.Fatalf("/stop: handled=%v reply=%q", handled, reply)
	}
	_ = sendSignal(t, svc)
	if len(adapter.signals) != 0 {
		t.Fatalf("signal after /stop must not open trades, got %d", len(adapter.signals))
	}

	reply, handled = command(svc, "/start")
	if !handled || !strings.Contains(reply, "включена") {
		t.Fatalf("/start: handled=%v reply=%q", handled, reply)
	}
	_ = sendSignal(t, svc)
	if len(adapter.signals) != 3 {
		t.Fatalf("signal after /start must open 3 trades, got %d", len(adapter.signals))
	}
}

func TestCommandRepliesWhenAlreadyInState(t *testing.T) {
	svc, _ := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

	if reply, _ := command(svc, "/start"); !strings.Contains(reply, "уже включена") {
		t.Fatalf("got %q", reply)
	}
	command(svc, "/stop")
	if reply, _ := command(svc, "/stop"); !strings.Contains(reply, "уже остановлена") {
		t.Fatalf("got %q", reply)
	}
}

func TestCommandVariants(t *testing.T) {
	for _, text := range []string{"/stop", "/STOP", "  /stop  ", "/stop@trading_adapter_bot", "/stop now"} {
		t.Run(text, func(t *testing.T) {
			svc, ks := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

			if _, handled := command(svc, text); !handled {
				t.Fatal("must be handled as a command")
			}
			if stopped, _ := ks.Stopped(); !stopped {
				t.Fatal("must be stopped")
			}
		})
	}
}

func TestNonCommandsAreNotHandled(t *testing.T) {
	svc, ks := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

	for _, text := range []string{validText, "hello", "", "/unknown", "stop", "please /stop"} {
		if _, handled := command(svc, text); handled {
			t.Fatalf("%q must not be handled as a command", text)
		}
	}
	if stopped, _ := ks.Stopped(); stopped {
		t.Fatal("non-commands must not stop trading")
	}
}

func TestCommandsFromOtherChatAreIgnored(t *testing.T) {
	svc, ks := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

	reply, handled := svc.HandleCommand(context.Background(), testChatID+1, "/stop", time.Now())
	if handled || reply != "" {
		t.Fatalf("handled=%v reply=%q, want false and empty", handled, reply)
	}
	if stopped, _ := ks.Stopped(); stopped {
		t.Fatal("a stranger must not be able to stop trading")
	}

	_ = ks.Stop(killswitch.ReasonManual)
	svc.HandleCommand(context.Background(), testChatID+1, "/start", time.Now())
	if stopped, _ := ks.Stopped(); !stopped {
		t.Fatal("a stranger must not be able to resume trading")
	}
}

func TestStaleStartCommandIsIgnored(t *testing.T) {
	svc, ks := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})
	_ = ks.Stop(killswitch.ReasonManual)

	reply, handled := svc.HandleCommand(context.Background(), testChatID, "/start", time.Now().Add(-2*testMaxAge))
	if !handled || reply != "" {
		t.Fatalf("stale /start: handled=%v reply=%q, want handled and silent", handled, reply)
	}
	if stopped, _ := ks.Stopped(); !stopped {
		t.Fatal("a stale /start from the backlog must not resume trading")
	}
}

func TestStaleStopCommandIsStillApplied(t *testing.T) {
	svc, ks := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

	svc.HandleCommand(context.Background(), testChatID, "/stop", time.Now().Add(-time.Hour))
	if stopped, _ := ks.Stopped(); !stopped {
		t.Fatal("/stop is always safe to apply, even when stale")
	}
}

func TestStatusCommand(t *testing.T) {
	svc, _ := newTestService(t, &mt5AdapterStub{}, killswitch.Config{MaxConsecutiveErrors: 5, MaxSignalsPerDay: 20})

	reply, handled := command(svc, "/status")
	if !handled {
		t.Fatal("must be handled")
	}
	for _, want := range []string{"включена", "0/5", "0/20"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status %q must contain %q", reply, want)
		}
	}

	_ = sendSignal(t, svc)
	command(svc, "/stop")
	reply, _ = command(svc, "/status")
	for _, want := range []string{"остановлена", "вручную", "1/20"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("status %q must contain %q", reply, want)
		}
	}
}

func TestConsecutiveErrorsStopTradingMidSignal(t *testing.T) {
	openErr := errors.New("mt5 adapter down")
	adapter := &mt5AdapterStub{err: openErr}
	svc, ks := newTestService(t, adapter, killswitch.Config{MaxConsecutiveErrors: 2})

	err := sendSignal(t, svc)
	if !errors.Is(err, openErr) {
		t.Fatalf("got %v, want %v", err, openErr)
	}
	// стоп сработал после второй ошибки, третья сделка уже не пробуется
	if len(adapter.signals) != 2 {
		t.Fatalf("OpenOrder calls: got %d, want 2", len(adapter.signals))
	}
	if stopped, reason := ks.Stopped(); !stopped || !strings.Contains(reason, "ошибок подряд") {
		t.Fatalf("stopped=%v reason=%q", stopped, reason)
	}

	// следующие сигналы блокируются
	adapter.signals = nil
	_ = sendSignal(t, svc)
	if len(adapter.signals) != 0 {
		t.Fatalf("signal while stopped must not open trades, got %d", len(adapter.signals))
	}
}

func TestSignalsPerDayLimitStopsTrading(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc, ks := newTestService(t, adapter, killswitch.Config{MaxSignalsPerDay: 2})

	_ = sendSignal(t, svc)
	_ = sendSignal(t, svc)
	if len(adapter.signals) != 6 {
		t.Fatalf("first two signals must open 3 trades each, got %d", len(adapter.signals))
	}

	_ = sendSignal(t, svc) // третий сигнал за сутки
	if len(adapter.signals) != 6 {
		t.Fatalf("third signal must be blocked, got %d trades in total", len(adapter.signals))
	}
	if stopped, reason := ks.Stopped(); !stopped || !strings.Contains(reason, "лимит сигналов") {
		t.Fatalf("stopped=%v reason=%q", stopped, reason)
	}
}

func TestInvalidMessagesDoNotCountAsSignals(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc, ks := newTestService(t, adapter, killswitch.Config{MaxSignalsPerDay: 1})

	for i := 0; i < 5; i++ {
		_ = svc.HandleMessage(context.Background(), testChatID, "просто болтовня", time.Now())
	}
	if got := ks.Status().SignalsLast24h; got != 0 {
		t.Fatalf("chatter must not count as signals, got %d", got)
	}
}

func TestAutoStopByErrorsNotifiesChat(t *testing.T) {
	adapter := &mt5AdapterStub{err: errors.New("mt5 adapter http 502: account mode real is not allowed")}
	svc, _ := newTestService(t, adapter, killswitch.Config{MaxConsecutiveErrors: 2})

	_ = sendSignal(t, svc)

	msgs := sentMessages(svc)
	if len(msgs) != 1 {
		t.Fatalf("expected exactly one notification, got %d: %q", len(msgs), msgs)
	}
	for _, want := range []string{"Автостоп", "ошибок подряд: 2", "account mode real is not allowed", "/start"} {
		if !strings.Contains(msgs[0], want) {
			t.Fatalf("notification %q must contain %q", msgs[0], want)
		}
	}

	// пока остановлено, повторных уведомлений нет
	_ = sendSignal(t, svc)
	if got := len(sentMessages(svc)); got != 1 {
		t.Fatalf("no repeated notifications while stopped, got %d", got)
	}
}

func TestAutoStopBySignalLimitNotifiesChat(t *testing.T) {
	svc, _ := newTestService(t, &mt5AdapterStub{}, killswitch.Config{MaxSignalsPerDay: 1})

	_ = sendSignal(t, svc)
	if got := len(sentMessages(svc)); got != 0 {
		t.Fatalf("no notification before the limit is hit, got %d", got)
	}

	_ = sendSignal(t, svc)
	msgs := sentMessages(svc)
	if len(msgs) != 1 {
		t.Fatalf("expected one notification, got %d", len(msgs))
	}
	for _, want := range []string{"Автостоп", "лимит сигналов за 24 ч: 1", "сигнал пропущен", "/start"} {
		if !strings.Contains(msgs[0], want) {
			t.Fatalf("notification %q must contain %q", msgs[0], want)
		}
	}
}

func TestManualStopDoesNotSendAutoStopNotification(t *testing.T) {
	svc, _ := newTestService(t, &mt5AdapterStub{}, killswitch.Config{})

	command(svc, "/stop")
	if got := len(sentMessages(svc)); got != 0 {
		t.Fatalf("manual /stop is answered via reply, not a notification; got %d", got)
	}
}

func TestNotificationFailureDoesNotBreakAutoStop(t *testing.T) {
	adapter := &mt5AdapterStub{}
	svc, ks := newTestService(t, adapter, killswitch.Config{MaxSignalsPerDay: 1})
	svc.notifier.(*notifierStub).err = errors.New("telegram unreachable")

	_ = sendSignal(t, svc)
	if err := sendSignal(t, svc); err != nil {
		t.Fatalf("notification failure must not surface as an error: %v", err)
	}
	if stopped, _ := ks.Stopped(); !stopped {
		t.Fatal("must be stopped even if the notification could not be sent")
	}
}

func TestAutoStopWithoutNotifierIsSafe(t *testing.T) {
	svc := NewService(&mt5AdapterStub{}, testChatID, testVolumes, testMaxAge, newKillSwitch(t, killswitch.Config{MaxSignalsPerDay: 1}), nil)

	_ = sendSignal(t, svc)
	_ = sendSignal(t, svc) // сработает автостоп, notifier == nil
}

func TestLastErrorNoteIsTruncated(t *testing.T) {
	note := lastErrorNote(errors.New(strings.Repeat("я", 1000)))
	if got := len([]rune(note)); got > len("Последняя ошибка: ")+maxErrorNoteLen+1 {
		t.Fatalf("note is too long: %d runes", got)
	}
	if lastErrorNote(nil) != "" {
		t.Fatal("no error — no note")
	}
}
