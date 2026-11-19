// Package killswitch — аварийный выключатель торговли.
//
// Остановленный trader не открывает новые сделки (уже открытые позиции не трогает).
// Остановить можно вручную или автоматически: по числу ошибок подряд и по лимиту
// сигналов за сутки. Состояние «остановлено» сохраняется в файл и переживает перезапуск.
package killswitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	signalsWindow = 24 * time.Hour

	ReasonManual = "вручную"
)

type Config struct {
	MaxConsecutiveErrors int    // стоп после N неудачных сделок подряд; 0 — не проверять
	MaxSignalsPerDay     int    // стоп, когда за 24 ч принято больше N сигналов; 0 — не проверять
	StateFile            string // файл состояния; "" — состояние хранится только в памяти
}

type Status struct {
	Stopped              bool
	Reason               string
	Since                time.Time
	ConsecutiveErrors    int
	SignalsLast24h       int
	MaxConsecutiveErrors int
	MaxSignalsPerDay     int
}

// state — то, что переживает перезапуск. Счётчики ошибок и сигналов только в памяти.
type state struct {
	Stopped bool      `json:"stopped"`
	Reason  string    `json:"reason"`
	Since   time.Time `json:"since"`
}

type Switch struct {
	cfg Config
	now func() time.Time

	mu                sync.Mutex
	st                state
	consecutiveErrors int
	signals           []time.Time
}

func New(cfg Config) (*Switch, error) {
	if cfg.MaxConsecutiveErrors < 0 || cfg.MaxSignalsPerDay < 0 {
		return nil, errors.New("killswitch: limits must be >= 0")
	}

	s := &Switch{cfg: cfg, now: time.Now}
	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// Stopped сообщает, остановлена ли торговля, и причину.
func (s *Switch) Stopped() (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.st.Stopped, s.st.Reason
}

// Stop останавливает торговлю. Если уже остановлена, причина остаётся первой.
// При ошибке сохранения торговля всё равно остановлена (в памяти).
func (s *Switch) Stop(reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.stopLocked(reason)

	return err
}

// Resume включает торговлю и обнуляет счётчики ошибок и сигналов.
// wasStopped — была ли торговля остановлена до вызова.
func (s *Switch) Resume() (wasStopped bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	wasStopped = s.st.Stopped
	s.st = state{}
	s.consecutiveErrors = 0
	s.signals = nil

	return wasStopped, s.save()
}

// AllowSignal вызывается один раз на каждый принятый сигнал. Возвращает false,
// если торговля остановлена или лимит сигналов за 24 часа исчерпан (тогда
// выключатель срабатывает, stoppedNow=true).
func (s *Switch) AllowSignal() (allowed, stoppedNow bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.st.Stopped {
		return false, false, nil
	}

	now := s.now()
	s.pruneSignals(now)

	if s.cfg.MaxSignalsPerDay > 0 && len(s.signals) >= s.cfg.MaxSignalsPerDay {
		stoppedNow, err = s.stopLocked(fmt.Sprintf("лимит сигналов за 24 ч: %d", s.cfg.MaxSignalsPerDay))
		return false, stoppedNow, err
	}

	s.signals = append(s.signals, now)

	return true, false, nil
}

// RecordTradeResult учитывает результат открытия сделки: успех сбрасывает счётчик
// ошибок подряд, неудача увеличивает его и может остановить торговлю.
func (s *Switch) RecordTradeResult(tradeErr error) (stoppedNow bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tradeErr == nil {
		s.consecutiveErrors = 0
		return false, nil
	}

	s.consecutiveErrors++
	if s.cfg.MaxConsecutiveErrors > 0 && s.consecutiveErrors >= s.cfg.MaxConsecutiveErrors {
		return s.stopLocked(fmt.Sprintf("ошибок подряд: %d", s.consecutiveErrors))
	}

	return false, nil
}

func (s *Switch) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneSignals(s.now())

	return Status{
		Stopped:              s.st.Stopped,
		Reason:               s.st.Reason,
		Since:                s.st.Since,
		ConsecutiveErrors:    s.consecutiveErrors,
		SignalsLast24h:       len(s.signals),
		MaxConsecutiveErrors: s.cfg.MaxConsecutiveErrors,
		MaxSignalsPerDay:     s.cfg.MaxSignalsPerDay,
	}
}

func (s *Switch) stopLocked(reason string) (stoppedNow bool, err error) {
	if s.st.Stopped {
		return false, nil
	}

	s.st = state{Stopped: true, Reason: reason, Since: s.now()}

	return true, s.save()
}

func (s *Switch) pruneSignals(now time.Time) {
	cutoff := now.Add(-signalsWindow)
	i := 0
	for i < len(s.signals) && s.signals[i].Before(cutoff) {
		i++
	}
	s.signals = s.signals[i:]
}

func (s *Switch) load() error {
	if s.cfg.StateFile == "" {
		return nil
	}

	data, err := os.ReadFile(s.cfg.StateFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("killswitch: read state file: %w", err)
	}

	if err := json.Unmarshal(data, &s.st); err != nil {
		return fmt.Errorf("killswitch: corrupt state file %s (fix or delete it): %w", s.cfg.StateFile, err)
	}

	return nil
}

// save пишет состояние атомарно: во временный файл и переименованием.
func (s *Switch) save() error {
	if s.cfg.StateFile == "" {
		return nil
	}

	data, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return fmt.Errorf("killswitch: encode state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.cfg.StateFile), 0o750); err != nil {
		return fmt.Errorf("killswitch: create state dir: %w", err)
	}

	tmp := s.cfg.StateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("killswitch: write state file: %w", err)
	}
	if err := os.Rename(tmp, s.cfg.StateFile); err != nil {
		return fmt.Errorf("killswitch: replace state file: %w", err)
	}

	return nil
}
