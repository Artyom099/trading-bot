package telegram_v1

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	entryRe = regexp.MustCompile(`(?i)зона входа:\s*([0-9]+(?:\.[0-9]+)?)`)
	tp1Re   = regexp.MustCompile(`(?i)tp1:\s*([0-9]+(?:\.[0-9]+)?)`)
	tp2Re   = regexp.MustCompile(`(?i)tp2:\s*([0-9]+(?:\.[0-9]+)?)`)
	tp3Re   = regexp.MustCompile(`(?i)tp3:\s*([0-9]+(?:\.[0-9]+)?)`)
	slRe    = regexp.MustCompile(`(?i)стоп:\s*([0-9]+(?:\.[0-9]+)?)`)
)

// tpRe — одиночная цель без номера ("→ TP: 3612.01"), считается первой целью, если нет явного TP1.
var tpRe = regexp.MustCompile(`(?i)\btp:\s*([0-9]+(?:\.[0-9]+)?)`)

// Decode разбирает plaintext Telegram-сообщение по контракту v1.
func Decode(text string) (Signal, error) {
	sig := Signal{
		Entry: parsePrice(entryRe, text),
		SL:    parsePrice(slRe, text),
		TP1:   parseFirstTarget(text),
		TP2:   parsePrice(tp2Re, text),
		TP3:   parsePrice(tp3Re, text),
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if sig.Side == "" {
			if side, ok := parseSide(line); ok {
				sig.Side = side
				continue
			}
		}

		if strings.Contains(line, "|") {
			if symbol := pickSymbol(line); symbol != "" {
				sig.Symbol = symbol
			}
		}
	}

	if sig.Side == "" {
		return Signal{}, ErrMissingSide
	}
	if sig.Symbol == "" {
		return Signal{}, ErrMissingSymbol
	}
	if sig.SL <= 0 {
		return Signal{}, ErrMissingSL
	}

	return sig, nil
}

func parseSide(raw string) (Side, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "buy", "long", "лонг":
		return SideBuy, true
	case "sell", "short", "шорт":
		return SideSell, true
	default:
		return "", false
	}
}

func pickSymbol(line string) string {
	// Берём первый ASCII-тикер слева: в "GOLD | ЗОЛОТО | XAUUSD" это GOLD — так символ называется у брокера.
	for _, part := range strings.Split(line, "|") {
		symbol := strings.ToUpper(strings.TrimSpace(part))
		if isTicker(symbol) {
			return symbol
		}
	}

	return ""
}

func isTicker(symbol string) bool {
	if len(symbol) < 3 {
		return false
	}

	for _, r := range symbol {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		if r > unicode.MaxASCII {
			return false
		}
	}

	return true
}

// parseFirstTarget возвращает TP1, а если его нет, одиночную цель "TP:" без номера. Явный TP1 приоритетнее.
func parseFirstTarget(text string) float64 {
	if price := parsePrice(tp1Re, text); price > 0 {
		return price
	}

	return parsePrice(tpRe, text)
}

func parsePrice(re *regexp.Regexp, text string) float64 {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return 0
	}

	price, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}

	return price
}
