package killswitch

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var errTrade = errors.New("trade failed")

func mustNew(t *testing.T, cfg Config) *Switch {
	t.Helper()

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return s
}

func TestStopAndResume(t *testing.T) {
	s := mustNew(t, Config{})

	if stopped, _ := s.Stopped(); stopped {
		t.Fatal("must be running by default")
	}
	if err := s.Stop(ReasonManual); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if stopped, reason := s.Stopped(); !stopped || reason != ReasonManual {
		t.Fatalf("got stopped=%v reason=%q", stopped, reason)
	}

	// повторный Stop не затирает первую причину
	if err := s.Stop("другая причина"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if _, reason := s.Stopped(); reason != ReasonManual {
		t.Fatalf("reason overwritten: %q", reason)
	}

	was, err := s.Resume()
	if err != nil || !was {
		t.Fatalf("Resume: was=%v err=%v", was, err)
	}
	if stopped, _ := s.Stopped(); stopped {
		t.Fatal("must be running after Resume")
	}
	if was, _ := s.Resume(); was {
		t.Fatal("second Resume must report wasStopped=false")
	}
}

func TestStatePersistsAcrossRestart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "killswitch.json")

	first := mustNew(t, Config{StateFile: file})
	if err := first.Stop("тест"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	second := mustNew(t, Config{StateFile: file})
	if stopped, reason := second.Stopped(); !stopped || reason != "тест" {
		t.Fatalf("state lost after restart: stopped=%v reason=%q", stopped, reason)
	}

	if _, err := second.Resume(); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	third := mustNew(t, Config{StateFile: file})
	if stopped, _ := third.Stopped(); stopped {
		t.Fatal("Resume must be persisted too")
	}
}

func TestMissingStateFileMeansRunning(t *testing.T) {
	s := mustNew(t, Config{StateFile: filepath.Join(t.TempDir(), "none.json")})
	if stopped, _ := s.Stopped(); stopped {
		t.Fatal("must be running when no state file")
	}
}

func TestCorruptStateFileIsAnError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "killswitch.json")
	if err := os.WriteFile(file, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := New(Config{StateFile: file}); err == nil {
		t.Fatal("corrupt state file must fail loudly, not silently enable trading")
	}
}

func TestNegativeLimitsRejected(t *testing.T) {
	if _, err := New(Config{MaxConsecutiveErrors: -1}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := New(Config{MaxSignalsPerDay: -1}); err == nil {
		t.Fatal("expected error")
	}
}

func TestConsecutiveErrorsStop(t *testing.T) {
	s := mustNew(t, Config{MaxConsecutiveErrors: 3})

	for i := 1; i <= 2; i++ {
		if stoppedNow, _ := s.RecordTradeResult(errTrade); stoppedNow {
			t.Fatalf("stopped too early at error %d", i)
		}
	}
	if stoppedNow, _ := s.RecordTradeResult(errTrade); !stoppedNow {
		t.Fatal("must stop at the 3rd consecutive error")
	}
	if stopped, reason := s.Stopped(); !stopped || reason == "" {
		t.Fatalf("stopped=%v reason=%q", stopped, reason)
	}
}

func TestSuccessResetsConsecutiveErrors(t *testing.T) {
	s := mustNew(t, Config{MaxConsecutiveErrors: 3})

	s.RecordTradeResult(errTrade)
	s.RecordTradeResult(errTrade)
	s.RecordTradeResult(nil)
	s.RecordTradeResult(errTrade)
	s.RecordTradeResult(errTrade)

	if stopped, _ := s.Stopped(); stopped {
		t.Fatal("success in the middle must reset the counter")
	}
}

func TestZeroLimitsDisableChecks(t *testing.T) {
	s := mustNew(t, Config{})

	for i := 0; i < 100; i++ {
		s.RecordTradeResult(errTrade)
		if allowed, _, _ := s.AllowSignal(); !allowed {
			t.Fatal("limits are disabled, signal must be allowed")
		}
	}
	if stopped, _ := s.Stopped(); stopped {
		t.Fatal("must not stop when limits are 0")
	}
}

func TestSignalsPerDayLimit(t *testing.T) {
	s := mustNew(t, Config{MaxSignalsPerDay: 2})

	for i := 0; i < 2; i++ {
		if allowed, _, _ := s.AllowSignal(); !allowed {
			t.Fatalf("signal %d must be allowed", i+1)
		}
	}

	allowed, stoppedNow, _ := s.AllowSignal()
	if allowed || !stoppedNow {
		t.Fatalf("3rd signal: allowed=%v stoppedNow=%v, want false/true", allowed, stoppedNow)
	}
	if stopped, _ := s.Stopped(); !stopped {
		t.Fatal("must be stopped after exceeding the limit")
	}

	// пока остановлено — сигналы блокируются, но «stoppedNow» уже не срабатывает
	allowed, stoppedNow, _ = s.AllowSignal()
	if allowed || stoppedNow {
		t.Fatalf("while stopped: allowed=%v stoppedNow=%v, want false/false", allowed, stoppedNow)
	}
}

func TestSignalsWindowExpires(t *testing.T) {
	s := mustNew(t, Config{MaxSignalsPerDay: 2})
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	s.AllowSignal()
	s.AllowSignal()

	now = now.Add(25 * time.Hour) // старые сигналы вышли из окна

	if allowed, _, _ := s.AllowSignal(); !allowed {
		t.Fatal("signals older than 24h must not count")
	}
	if got := s.Status().SignalsLast24h; got != 1 {
		t.Fatalf("SignalsLast24h: got %d, want 1", got)
	}
}

func TestResumeResetsCounters(t *testing.T) {
	s := mustNew(t, Config{MaxConsecutiveErrors: 2, MaxSignalsPerDay: 1})

	s.AllowSignal()
	s.RecordTradeResult(errTrade)
	s.RecordTradeResult(errTrade) // стоп по ошибкам

	if _, err := s.Resume(); err != nil {
		t.Fatal(err)
	}

	st := s.Status()
	if st.ConsecutiveErrors != 0 || st.SignalsLast24h != 0 {
		t.Fatalf("counters must be reset: %+v", st)
	}
	if allowed, _, _ := s.AllowSignal(); !allowed {
		t.Fatal("after Resume the next signal must be allowed, not re-stopped by old counters")
	}
}

func TestStopInMemoryEvenIfSaveFails(t *testing.T) {
	// каталог состояния — это файл, поэтому создать файл состояния нельзя
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s := mustNew(t, Config{})
	s.cfg.StateFile = filepath.Join(blocker, "killswitch.json") // сохранение уже после успешного старта

	if err := s.Stop(ReasonManual); err == nil {
		t.Fatal("expected save error")
	}
	if stopped, _ := s.Stopped(); !stopped {
		t.Fatal("safety first: must be stopped in memory even when persisting fails")
	}
}

func TestConcurrentUse(t *testing.T) {
	s := mustNew(t, Config{MaxConsecutiveErrors: 1000, MaxSignalsPerDay: 1000})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.AllowSignal()
			s.RecordTradeResult(errTrade)
			s.Status()
			s.Stopped()
		}()
	}
	wg.Wait()
}
