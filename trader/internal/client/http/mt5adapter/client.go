package mt5adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Artyom099/trading-bot/platform/pkg/logger"
	"github.com/Artyom099/trading-bot/trader/internal/model"
)

type client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type orderRequest struct {
	Side    string  `json:"side"`
	Symbol  string  `json:"symbol"`
	Volume  float64 `json:"volume"`
	SL      float64 `json:"sl,omitempty"`
	TP      float64 `json:"tp,omitempty"`
	Comment string  `json:"comment,omitempty"`
}

func (c *client) OpenOrder(ctx context.Context, signal model.Signal) (err error) {
	payload, err := json.Marshal(orderRequest{
		Side:    string(signal.Side),
		Symbol:  signal.Symbol,
		Volume:  signal.Volume,
		SL:      signal.SL,
		TP:      signal.TP,
		Comment: signal.Comment,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/orders", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mt5 adapter request: %w", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if err == nil {
			err = closeErr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mt5 adapter http %d: %s", resp.StatusCode, body)
	}

	logger.Info(ctx, "mt5 adapter response", zap.ByteString("body", body))

	return nil
}
