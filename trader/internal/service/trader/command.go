package trader

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Artyom099/trading-bot/platform/pkg/logger"
	"github.com/Artyom099/trading-bot/trader/internal/killswitch"
)

func (s *service) HandleCommand(ctx context.Context, chatID int64, text string, sentAt time.Time) (string, bool) {
	if chatID != s.chatID {
		return "", false
	}

	switch parseCommand(text) {
	case "/stop":
		return s.stop(ctx), true
	case "/start":
		return s.start(ctx, sentAt), true
	case "/status":
		return s.status(), true
	default:
		return "", false
	}
}

func (s *service) stop(ctx context.Context) string {
	if stopped, reason := s.killSwitch.Stopped(); stopped {
		return fmt.Sprintf("⛔ Торговля уже остановлена (%s). Включить: /start", reason)
	}

	if err := s.killSwitch.Stop(killswitch.ReasonManual); err != nil {
		logger.Error(ctx, "failed to persist kill switch state", zap.Error(err))
	}
	logger.Warn(ctx, "⛔ trading stopped by command")

	return "⛔ Торговля остановлена. Новые сделки не открываются, открытые позиции не тронуты. Включить: /start"
}

// start игнорирует устаревшую команду: после простоя trader получает накопившиеся
// сообщения, и старый /start не должен неожиданно включить торговлю.
func (s *service) start(ctx context.Context, sentAt time.Time) string {
	if age := time.Since(sentAt); s.maxSignalAge > 0 && age > s.maxSignalAge {
		logger.Warn(ctx, "stale /start ignored", zap.Duration("age", age.Round(time.Second)))
		return ""
	}

	was, err := s.killSwitch.Resume()
	if err != nil {
		logger.Error(ctx, "failed to persist kill switch state", zap.Error(err))
	}
	if !was {
		return "✅ Торговля уже включена."
	}
	logger.Info(ctx, "✅ trading resumed by command")

	return "✅ Торговля включена."
}

func (s *service) status() string {
	st := s.killSwitch.Status()

	var b strings.Builder
	if st.Stopped {
		fmt.Fprintf(&b, "Торговля: ⛔ остановлена (%s, с %s)\n", st.Reason, st.Since.Local().Format("2006-01-02 15:04:05"))
	} else {
		b.WriteString("Торговля: ✅ включена\n")
	}
	fmt.Fprintf(&b, "Ошибок подряд: %s\n", counter(st.ConsecutiveErrors, st.MaxConsecutiveErrors))
	fmt.Fprintf(&b, "Сигналов за 24 ч: %s", counter(st.SignalsLast24h, st.MaxSignalsPerDay))

	return b.String()
}

func counter(current, limit int) string {
	if limit == 0 {
		return fmt.Sprintf("%d (без лимита)", current)
	}

	return fmt.Sprintf("%d/%d", current, limit)
}

// parseCommand возвращает команду из первого слова сообщения: "/stop@my_bot arg" -> "/stop".
// Для обычного текста возвращает пустую строку.
func parseCommand(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return ""
	}

	cmd, _, _ := strings.Cut(strings.ToLower(fields[0]), "@")

	return cmd
}
