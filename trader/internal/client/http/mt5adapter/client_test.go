package mt5adapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Artyom099/trading-bot/trader/internal/model"
)

func TestOpenOrderSuccess(t *testing.T) {
	var gotMethod, gotPath, gotCT string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")

		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL + "/")
	err := c.OpenOrder(context.Background(), model.Signal{
		Side:   model.SideSell,
		Symbol: "XAUUSD",
		Volume: 0.01,
		Entry:  4167.55,
		SL:     4175.89,
		TP:     4163.38,
		TP2:    4159.21,
		TP3:    4155.05,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("method: got %s, want POST", gotMethod)
	}
	if gotPath != "/v1/orders" {
		t.Fatalf("path: got %s, want /v1/orders", gotPath)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type: got %s, want application/json", gotCT)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	want := map[string]any{
		"side":   "sell",
		"symbol": "XAUUSD",
		"volume": 0.01,
		"sl":     4175.89,
		"tp":     4163.38,
	}
	if len(payload) != len(want) {
		t.Fatalf("json keys: got %v, want %v", payload, want)
	}
	for k, v := range want {
		if payload[k] != v {
			t.Fatalf("json %s: got %#v, want %#v", k, payload[k], v)
		}
	}
}

func TestOpenOrderHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"no tick for XAUUSD"}`))
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL)
	err := c.OpenOrder(context.Background(), model.Signal{
		Side:   model.SideBuy,
		Symbol: "XAUUSD",
		Volume: 0.01,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("error must contain status, got %v", err)
	}
	if !strings.Contains(err.Error(), "no tick for XAUUSD") {
		t.Fatalf("error must contain response body, got %v", err)
	}
}

func TestOpenOrderSendsComment(t *testing.T) {
	var payload map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("invalid json: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	err := NewClient(srv.URL).OpenOrder(context.Background(), model.Signal{
		Side: model.SideBuy, Symbol: "XAUUSD", Volume: 0.02, SL: 1, TP: 2, Comment: "tp2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload["comment"] != "tp2" {
		t.Fatalf("comment: got %#v, want tp2", payload["comment"])
	}
}
