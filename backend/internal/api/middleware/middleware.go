// Package middleware contains middleware functions for the API
package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/httplog/v3"
	"github.com/golang-jwt/jwt/v5"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/env"
	wcJwt "github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/role"

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

func AuthorizeRequest(requiredRole role.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			env := env.EnvFromCtx(r.Context())
			requestID := fmt.Sprintf("%d", extractRequestID(r.Context()))

			accessToken, err := r.Cookie(token.AccessTokenName(env))
			if err != nil {
				env.Logger.ErrorContext(r.Context(), "unable to get access token", slog.Any("error", err))
				_ = apiError.EncodeError(w, apiError.InvalidToken, "invalid access token", requestID)
				return
			}

			secret := env.Get("APP_SECRET")
			if secret == "" {
				env.Logger.ErrorContext(r.Context(), "environment variable APP_SECRET not set")
				_ = apiError.EncodeInternalError(w, requestID)
				return
			}
			secretVersion := env.Get("APP_SECRET_VERSION")
			if secretVersion == "" {
				env.Logger.ErrorContext(r.Context(), "environment variable APP_SECRET_VERSION not set")
				_ = apiError.EncodeInternalError(w, requestID)
				return
			}

			token, err := wcJwt.ValidateJWT(accessToken.Value, secretVersion, []byte(secret))
			if errors.Is(err, jwt.ErrTokenExpired) {
				env.Logger.ErrorContext(r.Context(), "access token expired", slog.Any("err", err))
				_ = apiError.EncodeError(w, apiError.ExpiredToken, "access token expired", requestID)
			} else if err != nil {
				env.Logger.ErrorContext(r.Context(), "invalid access token", slog.Any("error", err))
				apiError.EncodeError(w, apiError.InvalidToken, "invalid access token", requestID)
				return
			}

			sub, _ := token.Claims.GetSubject()
			r = r.WithContext(log.AppendCtx(r.Context(), slog.String("user-id", sub)))
			env.Logger.DebugContext(r.Context(), "validating user role")

			roleClaim := token.Claims.(jwt.MapClaims)["role"].(string)
			userRole := role.ToRole(roleClaim)
			if userRole < requiredRole {
				apiError.EncodeError(w, apiError.InsufficientPermissions, "insufficient permissions", requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func injectRequestID(ctx context.Context, requestID uint64) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func extractRequestID(ctx context.Context) uint64 {
	if v, ok := ctx.Value(requestIDKey).(uint64); ok {
		return v
	}
	return 0
}
