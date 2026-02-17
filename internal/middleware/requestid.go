package middleware

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/devaloi/shrink/internal/encoding"
)

type contextKey string

const requestIDKey contextKey = "requestID"

var counter int64

// RequestID adds a unique request ID to each request via X-Request-ID header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			n := atomic.AddInt64(&counter, 1)
			requestID = encoding.Encode(n)
		}

		w.Header().Set("X-Request-ID", requestID)

		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
