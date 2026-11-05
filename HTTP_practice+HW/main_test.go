package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	h := newRouter(NewStore())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type=%q", rec.Header().Get("Content-Type"))
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status in body=%q", body.Status)
	}
}

func TestCreateOrder(t *testing.T) {
	store := NewStore()
	h := newRouter(store)
	ids := make(map[int]bool)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"customer":"  Анна  "}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status=%d, want 201; body=%s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type=%q", rec.Header().Get("Content-Type"))
		}
		var order Order
		if err := json.Unmarshal(rec.Body.Bytes(), &order); err != nil {
			t.Fatal(err)
		}
		if order.ID <= 0 || order.Customer != "Анна" || order.Status != "new" {
			t.Fatalf("unexpected order: %+v", order)
		}
		if ids[order.ID] {
			t.Fatalf("duplicate ID: %d", order.ID)
		}
		ids[order.ID] = true
		if saved, ok := store.ByID(order.ID); !ok || saved != order {
			t.Fatalf("stored order=%+v, found=%v", saved, ok)
		}
	}
}

func TestCreateOrderInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"whitespace body", " \n\t"},
		{"broken JSON", `{"customer":`},
		{"wrong customer type", `{"customer":123}`},
		{"missing customer", `{}`},
		{"empty name", `{"customer":""}`},
		{"whitespace name", `{"customer":" \t\n "}`},
		{"null", `null`},
		{"null customer", `{"customer":null}`},
		{"array", `[]`},
		{"second JSON", `{"customer":"Анна"} {}`},
		{"trailing garbage", `{"customer":"Анна"} text`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore()
			existing := store.Create("Борис")
			h := newRouter(store)
			req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status=%d, want 400", rec.Code)
			}
			if rec.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type=%q", rec.Header().Get("Content-Type"))
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error == "" {
				t.Fatal("empty error message")
			}
			if len(store.orders) != 1 || store.nextID != 2 || store.orders[existing.ID] != existing {
				t.Fatal("invalid request changed the store")
			}
		})
	}
}

func TestAccessLog(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	h := newRouter(NewStore())

	for _, tt := range []struct {
		body   string
		status int
	}{
		{"{\"customer\":\"Анна\"} \n\t", http.StatusCreated},
		{`{"customer":" "}`, http.StatusBadRequest},
	} {
		output.Reset()
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var entry map[string]any
		if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tt.status || entry["status"] != float64(tt.status) {
			t.Fatalf("response=%d, logged status=%v, want %d", rec.Code, entry["status"], tt.status)
		}
		if entry["method"] != "POST" || entry["path"] != "/orders" {
			t.Fatalf("unexpected log: %v", entry)
		}
		if duration, ok := entry["duration_ms"].(float64); !ok || duration < 0 {
			t.Fatalf("invalid duration_ms: %v", entry["duration_ms"])
		}
		if strings.Contains(output.String(), "Анна") || strings.Contains(output.String(), "customer") {
			t.Fatal("log contains customer data")
		}
	}
}
