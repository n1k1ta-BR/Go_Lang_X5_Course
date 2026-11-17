package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"example.com/lesson9practice/internal/model"
)

func TestListOrdersPagination(t *testing.T) {
	handler := newHandler()

	for i := 0; i < 25; i++ {
		w := request(
			handler, "POST", "/api/v1/orders",
			`{"customer":"Алиса"}`, "application/json",
		)
		if w.Code != 201 {
			t.Fatalf("create status=%d body=%s", w.Code, w.Body)
		}
	}

	maxInt := int(^uint(0) >> 1)

	tests := []struct {
		query string
		first int
		count int
	}{
		{"", 1, 20},
		{"?offset=20", 21, 5},
		{"?limit=1&offset=1", 2, 1},
		{"?limit=10", 1, 10},
		{"?limit=10&offset=10", 11, 10},
		{"?limit=10&offset=20", 21, 5},
		{"?limit=100", 1, 25},
		{"?limit=01&offset=01", 2, 1},
		{"?offset=25", 0, 0},
		{"?offset=26", 0, 0},
		{"?limit=100&offset=" + strconv.Itoa(maxInt), 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			w := request(
				handler, "GET", "/api/v1/orders"+tt.query, "", "",
			)
			if w.Code != 200 ||
				w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}

			var orders []OrderResponse
			if err := json.Unmarshal(w.Body.Bytes(), &orders); err != nil {
				t.Fatal(err)
			}
			if orders == nil || len(orders) != tt.count {
				t.Fatalf("want %d orders, got %s", tt.count, w.Body)
			}

			for i, order := range orders {
				if order.ID != tt.first+i ||
					order.Customer != "Алиса" ||
					order.Status != "new" {
					t.Fatalf("unexpected order %+v", order)
				}
			}
		})
	}
}

func TestListOrdersInvalidQuery(t *testing.T) {
	maxInt := strconv.Itoa(int(^uint(0) >> 1))

	queries := []string{
		"limit",
		"limit=",
		"limit=abc",
		"limit=-1",
		"limit=0",
		"limit=101",
		"limit=1.5",
		"limit=1e2",
		"limit=0x10",
		"limit=1_0",
		"limit=+1",
		"limit=%2B1",
		"limit=%201",
		"limit=1%20",
		"limit=%D9%A1",
		"limit=" + maxInt + "0",
		"limit=1&limit=2",
		"limit=1&limit=1",
		"limit=&limit=2",
		"offset",
		"offset=",
		"offset=abc",
		"offset=-1",
		"offset=%2B1",
		"offset=1.0",
		"offset=1e1",
		"offset=0x10",
		"offset=1_0",
		"offset=1%20",
		"offset=" + maxInt + "0",
		"offset=1&offset=2",
		"unknown=1",
		"Limit=1",
		"limit=1&unknown=1",
		"status=",
		"status=paid",
		"status=NEW",
		"status=%20new",
		"status=new&status=new",
		"limit=%zz",
		"offset=%",
		"limit=1;offset=2",
	}

	handler := newHandler()

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/orders", nil)
			r.URL.RawQuery = query

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			if w.Code != 400 ||
				w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}

			var out ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if out.Code != "INVALID_QUERY" || out.Error == "" {
				t.Fatalf("unexpected error %+v", out)
			}
		})
	}
}

func TestStatusFilterBeforePagination(t *testing.T) {
	handler := newHandler()

	for i := 0; i < 3; i++ {
		w := request(
			handler, "POST", "/api/v1/orders",
			`{"customer":"Алиса"}`, "application/json",
		)
		if w.Code != 201 {
			t.Fatalf("create status=%d body=%s", w.Code, w.Body)
		}
	}

	empty := request(
		handler, "GET", "/api/v1/orders?status=cancelled", "", "",
	)
	if empty.Code != 200 || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("status=%d body=%s", empty.Code, empty.Body)
	}

	for _, id := range []int{1, 3} {
		w := request(
			handler,
			"PATCH",
			"/api/v1/orders/"+strconv.Itoa(id)+"/status",
			`{"status":"cancelled"}`,
			"application/json",
		)
		if w.Code != 200 {
			t.Fatalf("cancel status=%d body=%s", w.Code, w.Body)
		}
	}

	for _, tt := range []struct {
		query  string
		id     int
		status string
	}{
		{"status=cancelled&limit=1&offset=1", 3, "cancelled"},
		{"status=new&limit=1", 2, "new"},
	} {
		w := request(
			handler, "GET", "/api/v1/orders?"+tt.query, "", "",
		)
		if w.Code != 200 {
			t.Fatalf("list status=%d body=%s", w.Code, w.Body)
		}

		var orders []OrderResponse
		if err := json.Unmarshal(w.Body.Bytes(), &orders); err != nil {
			t.Fatal(err)
		}
		if len(orders) != 1 ||
			orders[0].ID != tt.id ||
			orders[0].Status != tt.status {
			t.Fatalf("query=%s body=%s", tt.query, w.Body)
		}
	}
}

func TestCancelOrder(t *testing.T) {
	handler := newHandler()

	created := request(
		handler, "POST", "/api/v1/orders",
		`{"customer":"Алиса"}`, "application/json",
	)
	if created.Code != 201 {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body)
	}

	updated := request(
		handler, "PATCH", "/api/v1/orders/1/status",
		`{"status":"cancelled"}`, "application/json",
	)
	if updated.Code != 200 ||
		updated.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("cancel status=%d body=%s", updated.Code, updated.Body)
	}

	var order OrderResponse
	if err := json.Unmarshal(updated.Body.Bytes(), &order); err != nil {
		t.Fatal(err)
	}

	want := OrderResponse{ID: 1, Customer: "Алиса", Status: "cancelled"}
	if order != want {
		t.Fatalf("unexpected order %+v", order)
	}

	read := request(handler, "GET", "/api/v1/orders/1", "", "")
	if read.Code != 200 || read.Body.String() != updated.Body.String() {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body)
	}

	again := request(
		handler, "PATCH", "/api/v1/orders/1/status",
		`{"status":"cancelled"}`, "application/json",
	)

	var out ErrorResponse
	if err := json.Unmarshal(again.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if again.Code != 409 || out.Code != "ORDER_ALREADY_CANCELLED" {
		t.Fatalf("repeat status=%d body=%s", again.Code, again.Body)
	}
}

func TestCancelOrderErrors(t *testing.T) {
	tests := []struct {
		id, body, contentType string
		status                int
		code                  string
	}{
		{"abc", `{"status":"cancelled"}`, "application/json", 400, "INVALID_ID"},
		{"0", `{"status":"cancelled"}`, "application/json", 400, "INVALID_ID"},
		{"-1", `{"status":"cancelled"}`, "application/json", 400, "INVALID_ID"},
		{"999999999999999999999999", `{"status":"cancelled"}`, "application/json", 400, "INVALID_ID"},
		{"999", `{"status":"cancelled"}`, "application/json", 404, "ORDER_NOT_FOUND"},
		{"1", ``, "application/json", 400, "INVALID_JSON"},
		{"1", `null`, "application/json", 400, "INVALID_JSON"},
		{"1", `[]`, "application/json", 400, "INVALID_JSON"},
		{"1", `{"status":2}`, "application/json", 400, "INVALID_JSON"},
		{"1", `{"status":"cancelled","extra":1}`, "application/json", 400, "INVALID_JSON"},
		{"1", `{"status":"cancelled"} {}`, "application/json", 400, "INVALID_JSON"},
		{"1", `{}`, "application/json", 400, "VALIDATION_ERROR"},
		{"1", `{"status":""}`, "application/json", 400, "VALIDATION_ERROR"},
		{"1", `{"status":"new"}`, "application/json", 400, "VALIDATION_ERROR"},
		{"1", `{"status":"paid"}`, "application/json", 400, "VALIDATION_ERROR"},
		{"1", `{"status":"cancelled"}`, "text/plain", 415, "UNSUPPORTED_MEDIA_TYPE"},
		{"1", `{"status":"` + strings.Repeat("a", 1100) + `"}`, "application/json", 413, "BODY_TOO_LARGE"},
	}

	for _, tt := range tests {
		t.Run(tt.id+"/"+tt.body+"/"+tt.contentType, func(t *testing.T) {
			handler := newHandler()

			created := request(
				handler, "POST", "/api/v1/orders",
				`{"customer":"Алиса"}`, "application/json",
			)
			if created.Code != 201 {
				t.Fatalf("create status=%d", created.Code)
			}

			w := request(
				handler, "PATCH",
				"/api/v1/orders/"+tt.id+"/status",
				tt.body, tt.contentType,
			)

			var out ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if w.Code != tt.status ||
				out.Code != tt.code ||
				w.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}

			read := request(handler, "GET", "/api/v1/orders/1", "", "")

			var saved OrderResponse
			if err := json.Unmarshal(read.Body.Bytes(), &saved); err != nil {
				t.Fatal(err)
			}
			if read.Code != 200 || saved.Status != "new" {
				t.Fatalf("failed request changed order: %s", read.Body)
			}
		})
	}
}

func TestConcurrentCancelOrder(t *testing.T) {
	handler := newHandler()

	w := request(
		handler, "POST", "/api/v1/orders",
		`{"customer":"Алиса"}`, "application/json",
	)
	if w.Code != 201 {
		t.Fatalf("create status=%d", w.Code)
	}

	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 2)
	var workers sync.WaitGroup

	for i := 0; i < 2; i++ {
		workers.Add(1)

		go func() {
			defer workers.Done()
			<-start

			responses <- request(
				handler, "PATCH", "/api/v1/orders/1/status",
				`{"status":"cancelled"}`, "application/json",
			)
		}()
	}

	close(start)
	workers.Wait()
	close(responses)

	counts := make(map[int]int)

	for response := range responses {
		counts[response.Code]++

		if response.Code == 409 {
			var out ErrorResponse
			if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if out.Code != "ORDER_ALREADY_CANCELLED" {
				t.Fatalf("unexpected error %+v", out)
			}
		}
	}

	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent cancel responses=%v", counts)
	}
}

type failingOrders struct {
	Orders
}

func (failingOrders) List() ([]model.Order, error) {
	return nil, errors.New("storage unavailable")
}

func (failingOrders) UpdateStatus(int, string) (model.Order, error) {
	return model.Order{}, errors.New("storage unavailable")
}

func TestListAndCancelInternalErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := New(failingOrders{}, logger).Handler()

	for _, tt := range []struct {
		method, path, body, contentType string
	}{
		{"GET", "/api/v1/orders", "", ""},
		{"PATCH", "/api/v1/orders/1/status", `{"status":"cancelled"}`, "application/json"},
	} {
		w := request(handler, tt.method, tt.path, tt.body, tt.contentType)

		var out ErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if w.Code != 500 ||
			out.Code != "INTERNAL_ERROR" ||
			out.Error != "internal server error" {
			t.Fatalf("%s status=%d body=%s", tt.method, w.Code, w.Body)
		}
	}
}
