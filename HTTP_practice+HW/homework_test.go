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

func checkJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	if rec.Code != status {
		t.Fatalf("status=%d, want %d: %s", rec.Code, status, rec.Body)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected application/json")
	}
	if status < 400 {
		if !json.Valid(rec.Body.Bytes()) {
			t.Fatalf("invalid JSON: %s", rec.Body)
		}
		return
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Fatalf("expected a JSON error: %s", rec.Body)
	}
}

func TestReadOrder(t *testing.T) {
	h := newRouter(NewStore())
	create := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/orders", strings.NewReader(`{"customer":" Анна "}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	h.ServeHTTP(create, req)
	checkJSON(t, create, 201)
	location := create.Header().Get("Location")
	if location != "/orders/1" {
		t.Fatal("invalid Location")
	}
	read := httptest.NewRecorder()
	h.ServeHTTP(read, httptest.NewRequest("GET", location, nil))
	checkJSON(t, read, 200)
	var got Order
	if err := json.Unmarshal(read.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := Order{ID: 1, Customer: "Анна", Status: "new"}
	if got != want {
		t.Fatalf("order=%+v, want %+v", got, want)
	}
	createID := create.Header().Get("X-Request-ID")
	readID := read.Header().Get("X-Request-ID")
	if createID == "" || readID == "" || readID == createID {
		t.Fatal("expected different request IDs")
	}
}

func TestInvalidOrderID(t *testing.T) {
	h := newRouter(NewStore())
	for _, tc := range []struct {
		id     string
		status int
	}{
		{"999", 404}, {"abc", 400}, {"0", 400}, {"-1", 400}, {"999999999999999999999999999999", 400},
	} {
		t.Run(tc.id, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", "/orders/"+tc.id, nil))
			checkJSON(t, rec, tc.status)
		})
	}
}

func TestExtraBodyChecks(t *testing.T) {
	for _, tc := range []struct {
		name, body, ct string
		status         int
	}{
		{"missing media", `{"customer":"А"}`, "", 415},
		{"wrong media", `{"customer":"А"}`, "text/plain", 415},
		{"unknown", `{"customer":"А","age":20}`, "application/json", 400},
		{"client id", `{"customer":"А","id":42}`, "application/json", 400},
		{"client status", `{"customer":"А","status":"paid"}`, "application/json", 400},
		{"large body", `{"customer":"` + strings.Repeat("x", 1024) + `"}`, "application/json", 413},
		{"large suffix", `{"customer":"А"}` + strings.Repeat(" ", 1024), "application/json", 413},
		{"exact limit", `{"customer":"` + strings.Repeat("x", 1009) + `"}`, "application/json", 201},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/orders", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.ct)
			newRouter(store).ServeHTTP(rec, req)
			checkJSON(t, rec, tc.status)
			if tc.status != 201 && (len(store.orders) != 0 || store.nextID != 1) {
				t.Fatal("invalid input changed store")
			}
		})
	}
}

func TestRecoveryAndRequestID(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	var contextID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = requestIDFromContext(r.Context())
		panic("private client data")
	})
	h := requestID(accessLog(recovery(handler)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

	checkJSON(t, rec, http.StatusInternalServerError)
	if strings.TrimSpace(rec.Body.String()) != `{"error":"internal server error"}` {
		t.Fatalf("unexpected body: %s", rec.Body)
	}

	headerID := rec.Header().Get("X-Request-ID")
	if headerID == "" || contextID != headerID {
		t.Fatalf("context ID=%q, header ID=%q", contextID, headerID)
	}
	if strings.Contains(output.String(), "private client data") {
		t.Fatal("panic details leaked into the log")
	}

	// recovery пишет первую строку, accessLog — вторую.
	panicLog, accessLine, _ := strings.Cut(output.String(), "\n")
	idField := `"request_id":"` + headerID + `"`
	if !strings.Contains(panicLog, idField) || !strings.Contains(accessLine, idField) {
		t.Fatalf("expected the same request ID in both logs: %s", output.String())
	}
	if !strings.Contains(accessLine, `"status":500`) {
		t.Fatalf("expected status 500 in access log: %s", accessLine)
	}
}
