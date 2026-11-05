package main

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func newRouter(store *Store) http.Handler {
	r := chi.NewRouter()
	r.Use(requestID, accessLog, recovery)
	r.Get("/health", health)
	r.Post("/orders", createOrder(store))
	r.Get("/orders/{id}", getOrder(store))
	return r
}

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func createOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Customer string `json:"customer"`
		}

		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")) // проверка заголовка запроса (клиент отправляет JSON? + проверка ошибки)
		if err != nil || mediaType != "application/json" {
			writeJSON(w, r, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type должен быть application/json"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024) //ограничиваем размер тела запроса
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields() // мы не игнорируем ненужные поля, а отдаем 400 (не даем задать ползователю статус и тд самому)
		if err := decoder.Decode(&input); err != nil {
			writeInputError(w, r, err) // разные ошибки
			return
		}

		// если есть еще данные, кроме пробелов и EOF - это 400
		if err := decoder.Decode(new(any)); err != io.EOF {
			writeInputError(w, r, err)
			return
		}

		customer := strings.TrimSpace(input.Customer) // пробелы убираем
		if customer == "" {
			writeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "имя клиента не должно быть пустым"})
			return
		}

		// создаём заказ
		order := store.Create(customer)
		w.Header().Set("Location", "/orders/"+strconv.Itoa(order.ID))
		writeJSON(w, r, http.StatusCreated, order)
	}
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		slog.Error("encode response",
			"error", err,
			"request_id", requestIDFromContext(r.Context()),
		)
		status = http.StatusInternalServerError
		data = []byte(`{"error":"internal server error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(append(data, '\n')); err != nil {
		slog.Error("write response", "error", err, "request_id", requestIDFromContext(r.Context()))
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           newRouter(NewStore()),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func writeInputError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, r, http.StatusRequestEntityTooLarge, map[string]string{"error": "тело запроса больше 1024 байт"})
		return
	}
	writeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "ожидается один JSON-объект с полем customer типа string, без неизвестных полей"})
}

func getOrder(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(chi.URLParam(r, "id")) //айдишник берем из маршрута
		if err != nil || id <= 0 {
			writeJSON(w, r, http.StatusBadRequest, map[string]string{"error": "id должен быть положительным целым числом"})
			return
		}
		order, ok := store.ByID(id) // ok = true если заказ существует
		if !ok {
			writeJSON(w, r, http.StatusNotFound, map[string]string{"error": "заказ не найден"})
			return
		}
		writeJSON(w, r, http.StatusOK, order)
	}
}
