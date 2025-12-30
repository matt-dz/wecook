// Package middleware contains middleware functions for the API
package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
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
		hostOrigin := e.Get("HOST_ORIGIN")

		// Determine allowed origin based on the incoming Origin header
		var allowedOrigin string
		if e.IsProd() {
			allowedOrigin = hostOrigin
		} else if origin != "" {
			// In dev mode, allow all origins
			allowedOrigin = origin
		}

		if allowedOrigin == "" && hostOrigin != "" {
			// Fallback to HOST_ORIGIN if no matching origin
			allowedOrigin = hostOrigin
		}

		if allowedOrigin == "" {
			e.Logger.WarnContext(r.Context(),
				"HOST_ORIGIN not set and no valid origin found; Access-Control-Allow-Origin will be empty")
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

// Recoverer recovers from panics and returns a standardized error response.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if err, ok := rvr.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rvr)
				}

				e := env.EnvFromCtx(r.Context())
				requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))

				e.Logger.ErrorContext(r.Context(),
					"panic recovered",
					slog.Any("panic", rvr),
					slog.String("stack", string(debug.Stack())))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(&apiError.Error{
					Code:    apiError.InternalServerError,
					Status:  http.StatusInternalServerError,
					Message: "internal server error",
					ErrorID: requestID,
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func validateCSRFHeader(input *openapi3filter.AuthenticationInput) error {
	csrfHeader := input.RequestValidationInput.Request.Header.Get(token.CSRFTokenHeader)
	if csrfHeader == "" {
		return ErrMissingCSRFHeader
	}
	csrfCookie, err := input.RequestValidationInput.Request.Cookie(token.CSRFTokenName())
	if err != nil {
		return ErrMissingCSRFCookie
	}
	if subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) == 0 {
		return ErrCSRFTokenMismatch
	}

	return nil
}

// OAPIAuthFunc is the authentication function for oapi-codegen middleware.
func OAPIAuthFunc(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	// Extract security scheme name to determine required role
	var requiredRole role.Role
	switch input.SecuritySchemeName {
	case "AccessTokenUserBearer":
		requiredRole = role.RoleUser
	case "AccessTokenAdminBearer":
		requiredRole = role.RoleAdmin
	default:
		// No authentication required
		return nil
	}

	env := env.EnvFromCtx(ctx)
	requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(ctx))

	var accessToken string
	cookie, err := input.RequestValidationInput.Request.Cookie(token.AccessTokenName())
	if err != nil {
		env.Logger.DebugContext(ctx, "failed to get access token, searching for Bearer token next",
			slog.Any("error", err))
		authHeader := input.RequestValidationInput.Request.Header.Get(token.AuthorizationHeader)
		accessToken, err = token.ParseBearerToken(authHeader)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse authorization header", slog.Any("error", err))
			return &apiError.Error{
				Code:    apiError.InvalidCredentials,
				Status:  apiError.InvalidCredentials.StatusCode(),
				Message: "access token invalid or missing",
				ErrorID: requestID,
			}
		}
	} else {
		if slices.Contains(
			[]string{http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodDelete},
			input.RequestValidationInput.Request.Method) {
			// State-changing request - validate csrf tokens
			env.Logger.DebugContext(ctx, "validating csrf tokens")
			if err := validateCSRFHeader(input); err != nil {
				env.Logger.ErrorContext(ctx, "failed to validate csrf token", slog.Any("error", err))
				return &apiError.Error{
					Code:    apiError.InvalidCredentials,
					Status:  apiError.InvalidCredentials.StatusCode(),
					Message: err.Error(),
					ErrorID: requestID,
				}
			}
		}
		accessToken = cookie.Value
	}

	accessJwt, err := wcJwt.ValidateJWT(accessToken, env.Config.AppSecret.Version, []byte(*env.Config.AppSecret.Value))
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
	//   2. There was a validation error (400-level status)
	//   3. There was an internal server error

	requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))

	// 1. Error was returned from middleware
	var errBody *apiError.Error
	if errors.As(err, &errBody) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(opts.StatusCode)
		_ = json.NewEncoder(w).Encode(errBody) //nolint:errchkjson
		return
	}

	// 2. Validation error (use the status code from opts)
	if opts.StatusCode >= 400 && opts.StatusCode < 500 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(opts.StatusCode)
		_ = json.NewEncoder(w).Encode(&apiError.Error{ //nolint:errchkjson
			Code:    apiError.BadRequest,
			Status:  opts.StatusCode,
			Message: err.Error(),
			ErrorID: requestID,
		})
		return
	}

	// 3. An internal server error was surfaced
	_ = apiError.EncodeInternalError(w, requestID)
}
