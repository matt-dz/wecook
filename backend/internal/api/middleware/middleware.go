// Package middleware contains middleware functions for the API
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/httplog/v3"
	"github.com/golang-jwt/jwt/v5"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/env"
	wcJwt "github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/role"

	oapimw "github.com/oapi-codegen/nethttp-middleware"
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

// AuthorizeRequest creates a middleware that validates JWT tokens and checks user roles.
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

// OAPIAuthFunc is the authentication function for oapi-codegen middleware.
func OAPIAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// Extract security scheme name to determine required role
	var requiredRole role.Role
	switch input.SecuritySchemeName {
	case "AccessTokenAdmin":
		requiredRole = role.RoleAdmin
	case "AccessTokenUser":
		requiredRole = role.RoleUser
	default:
		// No authentication required
		return nil
	}

	env := env.EnvFromCtx(ctx)
	requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(ctx))

	accessToken, err := input.RequestValidationInput.Request.Cookie(token.AccessTokenName(env))
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get access token", slog.Any("error", err))
		return &apiError.Error{
			Code:    apiError.InvalidAccessToken,
			Status:  apiError.InvalidAccessToken.StatusCode(),
			Message: "invalid access token",
			ErrorID: requestID,
		}
	}

	secret := env.Get("APP_SECRET")
	if secret == "" {
		env.Logger.ErrorContext(ctx, "APP_SECRET not set")
		return &apiError.Error{
			Code:    apiError.InternalServerError,
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "internal server error",
			ErrorID: requestID,
		}
	}
	secretVersion := env.Get("APP_SECRET_VERSION")
	if secretVersion == "" {
		env.Logger.DebugContext(ctx, "APP_SECRET_VERSION not set, using default version")
		secretVersion = wcJwt.DefaultKID
	}

	accessJwt, err := wcJwt.ValidateJWT(accessToken.Value, secretVersion, []byte(secret))
	if errors.Is(err, jwt.ErrTokenExpired) {
		env.Logger.ErrorContext(ctx, "jwt expired", slog.Any("error", err))
		return &apiError.Error{
			Code:    apiError.ExpiredAccessToken,
			Status:  apiError.ExpiredAccessToken.StatusCode(),
			Message: "access token expired",
			ErrorID: requestID,
		}
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate jwt", slog.Any("error", err))
		return &apiError.Error{
			Code:    apiError.InvalidAccessToken,
			Status:  apiError.InvalidAccessToken.StatusCode(),
			Message: "invalid access token",
			ErrorID: requestID,
		}
	}

	sub, err := accessJwt.Claims.GetSubject()
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get subject", slog.Any("errro", err))
		return &apiError.Error{
			Code:    apiError.InternalServerError,
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "internal server error",
			ErrorID: requestID,
		}
	}
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to parse userID", slog.Any("error", err))
		return &apiError.Error{
			Code:    apiError.InternalServerError,
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "internal server error",
			ErrorID: requestID,
		}
	}

	roleClaim := accessJwt.Claims.(jwt.MapClaims)["role"].(string)
	userRole := role.ToRole(roleClaim)
	if userRole < requiredRole {
		env.Logger.ErrorContext(ctx, "user does not have required role",
			slog.String("user-role", userRole.String()),
			slog.String("required-role", requiredRole.String()))
		return &apiError.Error{
			Code:    apiError.InsufficientPermissions,
			Status:  apiError.InsufficientPermissions.StatusCode(),
			Message: "insufficient permissions",
			ErrorID: requestID,
		}
	}

	// Store user info in context
	r := input.RequestValidationInput.Request
	r = r.WithContext(log.AppendCtx(r.Context(), slog.Int64("user-id", userID)))
	r = r.WithContext(token.UserIDWithCtx(r.Context(), userID))
	r = r.WithContext(token.AccessTokenWithCtx(r.Context(), accessJwt))
	*input.RequestValidationInput.Request = *r

	return nil
}

// OAPIErrorHandler handles errors from oapi-codegen middleware and formats them
// according to your error schema.
func OAPIErrorHandler(
	ctx context.Context,
	err error,
	w http.ResponseWriter,
	r *http.Request,
	opts oapimw.ErrorHandlerOpts,
) {
	// Several scenarios where we are handling an error:
	//   1. An error was returned as an apiError in auth middleware
	//   2. There was an internal server error

	// 1. Error was returned from middleware
	var errBody *apiError.Error
	if errors.As(err, &errBody) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(opts.StatusCode)
		_ = json.NewEncoder(w).Encode(errBody)
		return
	}

	fmt.Printf("There is an error: %s", err.Error())
	// 2. An internal server error was surfaced
	requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))
	_ = apiError.EncodeInternalError(w, requestID)
}
