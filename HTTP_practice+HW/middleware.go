package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

type requestIDKey struct{}

var requestSequence atomic.Uint64

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d", requestSequence.Add(1))
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		// defer запишет в любом случае
		defer func() {
			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			slog.Info("http request",
				"request_id", requestIDFromContext(r.Context()),
				"method", r.Method, "path", r.URL.Path, "status", status,
				"duration_ms", float64(time.Since(start).Microseconds())/1000,
			)
		}()
		next.ServeHTTP(wrapped, r)
	})
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				if value == http.ErrAbortHandler {
					panic(value)
				}
				// значение паники не логируем, т. к. оно может содержать данные клиента
				slog.Error("panic recovered", "request_id", requestIDFromContext(r.Context()))
				// уже отправленный ответ заменить нельзя: прерываем его передачу
				if wrapped, ok := w.(middleware.WrapResponseWriter); ok && wrapped.Status() != 0 {
					panic(http.ErrAbortHandler)
				}
				writeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
