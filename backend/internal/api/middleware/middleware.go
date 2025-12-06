// Package middleware contains middleware functions for the API
package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/httplog/v3"
	"github.com/golang-jwt/jwt/v5"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
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
		r = r.WithContext(requestid.InjectRequestID(r.Context(), requestID))
		next.ServeHTTP(w, r)
	})
}

// AddCors adds the necessary CORS headers to the response.
func AddCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := env.EnvFromCtx(r.Context())
		origin := r.Header.Get("Origin")
		baseURL := e.Get("BASE_URL")
		isProd := e.Get("ENV") == "PROD"

		// Determine allowed origin based on the incoming Origin header
		var allowedOrigin string
		if isProd {
			allowedOrigin = baseURL
		} else if origin != "" {
			// In dev mode, allow all origins
			allowedOrigin = origin
		}

		if allowedOrigin == "" && baseURL != "" {
			// Fallback to BASE_URL if no matching origin
			allowedOrigin = baseURL
		}

		if allowedOrigin == "" {
			e.Logger.WarnContext(r.Context(),
				"BASE_URL not set and no valid origin found; Access-Control-Allow-Origin will be empty")
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthorizeRequest(requiredRole role.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			env := env.EnvFromCtx(r.Context())
			requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))

			accessToken, err := r.Cookie(token.AccessTokenName(env))
			if err != nil {
				env.Logger.ErrorContext(r.Context(), "unable to get access token", slog.Any("error", err))
				_ = apiError.EncodeError(w, apiError.InvalidAccessToken, "invalid access token", requestID)
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
				secretVersion = wcJwt.DefaultKID
			}

			accessJwt, err := wcJwt.ValidateJWT(accessToken.Value, secretVersion, []byte(secret))
			if errors.Is(err, jwt.ErrTokenExpired) {
				env.Logger.ErrorContext(r.Context(), "access token expired", slog.Any("err", err))
				_ = apiError.EncodeError(w, apiError.ExpiredAccessToken, "access token expired", requestID)
				return
			} else if err != nil {
				env.Logger.ErrorContext(r.Context(), "invalid access token", slog.Any("error", err))
				_ = apiError.EncodeError(w, apiError.InvalidAccessToken, "invalid access token", requestID)
				return
			}

			sub, err := accessJwt.Claims.GetSubject()
			if err != nil {
				env.Logger.ErrorContext(r.Context(), "failed to extract subject from jwt", slog.Any("error", err))
				_ = apiError.EncodeInternalError(w, requestID)
				return
			}
			userID, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				env.Logger.ErrorContext(r.Context(), "failed to parse user id", slog.Any("error", err))
				_ = apiError.EncodeInternalError(w, requestID)
				return
			}
			r = r.WithContext(log.AppendCtx(r.Context(), slog.Int64("user-id", userID)))
			r = r.WithContext(token.UserIDWithCtx(r.Context(), userID))
			env.Logger.DebugContext(r.Context(), "validating user role")

			roleClaim := accessJwt.Claims.(jwt.MapClaims)["role"].(string)
			userRole := role.ToRole(roleClaim)
			if userRole < requiredRole {
				_ = apiError.EncodeError(w, apiError.InsufficientPermissions, "insufficient permissions", requestID)
				return
			}
			r = r.WithContext(token.AccessTokenWithCtx(r.Context(), accessJwt))

			next.ServeHTTP(w, r)
		})
	}
}
