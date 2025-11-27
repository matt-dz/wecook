// Package middleware contains middleware functions for the API
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"

	"github.com/oklog/ulid/v2"
)

type requestIDKeyType struct{}

var requestIDKey requestIDKeyType

// InjectEnv injects an environment struct into the request context.
func InjectEnv(environment *env.Env) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(env.WithCtx(r.Context(), environment)))
		})
	}
}

func LogRequest(logger *slog.Logger) func(http.Handler) http.Handler {
	return httplog.RequestLogger(logger, &httplog.Options{
		LogExtraAttrs: func(r *http.Request, reqBody string, respStatus int) []slog.Attr {
			requestID := r.Context().Value(requestIDKey)
			if id, ok := requestID.(uint64); ok {
				return []slog.Attr{slog.Uint64("log_id", id)}
			}
			return []slog.Attr{slog.String("log_id", "N/A")}
		},
	})
}

// AddRequestID adds a request ID to the request context.
func AddRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := ulid.Now()
		r = r.WithContext(log.AppendCtx(r.Context(), slog.Uint64("log_id", requestID)))
		r = r.WithContext(injectRequestID(r.Context(), requestID))
		next.ServeHTTP(w, r)
	})
}

// AddCors adds the necessary CORS headers to the response.
func AddCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := env.EnvFromCtx(r.Context())
		if e.Get("ENV") == "PROD" {
			baseURL := e.Get("BASE_URL")
			if baseURL == "" {
				e.Logger.WarnContext(r.Context(), "BASE_URL not set; Access-Control-Allow-Origin will be empty")
			}
			w.Header().Add("Access-Control-Allow-Origin", baseURL)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Add("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		next.ServeHTTP(w, r)
	})
}

func injectRequestID(ctx context.Context, requestID uint64) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
