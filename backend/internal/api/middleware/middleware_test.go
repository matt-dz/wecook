package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/env"
	mJwt "github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/role"
)

func TestOAPIAuthFunc_CSRFTokenValidation(t *testing.T) {
	appSecret := "test-secret-32-bytes-long-12345"
	userID := int64(123)

	// Helper function to create a valid access token
	createAccessToken := func(t *testing.T, userRole role.Role) string {
		t.Helper()
		params := mJwt.JWTParams{
			UserID: fmt.Sprintf("%d", userID),
			Role:   userRole,
		}
		e := env.New(map[string]string{
			"APP_SECRET":         appSecret,
			"APP_SECRET_VERSION": "1",
		})
		accessToken, err := token.NewAccessToken(params, e)
		if err != nil {
			t.Fatalf("failed to create access token: %v", err)
		}
		return accessToken
	}

	tests := []struct {
		name               string
		securitySchemeName string
		setupRequest       func(*http.Request)
		wantError          bool
		wantErrorCode      apiError.ErrorCode
	}{
		{
			name:               "cookie auth with valid CSRF tokens",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				csrfToken, _ := token.NewCSRFToken()

				// Set access token cookie
				r.AddCookie(&http.Cookie{
					Name:  token.AccessTokenName(),
					Value: accessToken,
				})

				// Set matching CSRF token in cookie and header
				r.AddCookie(&http.Cookie{
					Name:  token.CSRFTokenName(),
					Value: csrfToken,
				})
				r.Header.Set(token.CSRFTokenHeader, csrfToken)
			},
			wantError: false,
		},
		{
			name:               "cookie auth with missing CSRF header",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				csrfToken, _ := token.NewCSRFToken()

				// Set access token cookie
				r.AddCookie(&http.Cookie{
					Name:  token.AccessTokenName(),
					Value: accessToken,
				})

				// Set CSRF cookie but NOT header
				r.AddCookie(&http.Cookie{
					Name:  token.CSRFTokenName(),
					Value: csrfToken,
				})
			},
			wantError:     true,
			wantErrorCode: apiError.InvalidCredentials,
		},
		{
			name:               "cookie auth with missing CSRF cookie",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				csrfToken, _ := token.NewCSRFToken()

				// Set access token cookie
				r.AddCookie(&http.Cookie{
					Name:  token.AccessTokenName(),
					Value: accessToken,
				})

				// Set CSRF header but NOT cookie
				r.Header.Set(token.CSRFTokenHeader, csrfToken)
			},
			wantError:     true,
			wantErrorCode: apiError.InvalidCredentials,
		},
		{
			name:               "cookie auth with mismatched CSRF tokens",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				csrfToken1, _ := token.NewCSRFToken()
				csrfToken2, _ := token.NewCSRFToken()

				// Set access token cookie
				r.AddCookie(&http.Cookie{
					Name:  token.AccessTokenName(),
					Value: accessToken,
				})

				// Set different CSRF tokens in cookie and header
				r.AddCookie(&http.Cookie{
					Name:  token.CSRFTokenName(),
					Value: csrfToken1,
				})
				r.Header.Set(token.CSRFTokenHeader, csrfToken2)
			},
			wantError:     true,
			wantErrorCode: apiError.InvalidCredentials,
		},
		{
			name:               "bearer token auth (no CSRF required)",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				// Use Authorization header instead of cookie - no CSRF needed
				r.Header.Set(token.AuthorizationHeader, "Bearer "+accessToken)
			},
			wantError: false,
		},
		{
			name:               "bearer token auth with invalid format",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				// Missing "Bearer " prefix
				r.Header.Set(token.AuthorizationHeader, accessToken)
			},
			wantError:     true,
			wantErrorCode: apiError.InvalidCredentials,
		},
		{
			name:               "no authentication required",
			securitySchemeName: "", // Empty security scheme
			setupRequest: func(r *http.Request) {
				// No auth needed for public endpoints
			},
			wantError: false,
		},
		{
			name:               "user role accessing user endpoint",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				r.Header.Set(token.AuthorizationHeader, "Bearer "+accessToken)
			},
			wantError: false,
		},
		{
			name:               "admin role accessing admin endpoint",
			securitySchemeName: "AccessTokenAdminBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleAdmin)
				r.Header.Set(token.AuthorizationHeader, "Bearer "+accessToken)
			},
			wantError: false,
		},
		{
			name:               "user role accessing admin endpoint - insufficient permissions",
			securitySchemeName: "AccessTokenAdminBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleUser)
				r.Header.Set(token.AuthorizationHeader, "Bearer "+accessToken)
			},
			wantError:     true,
			wantErrorCode: apiError.InsufficientPermissions,
		},
		{
			name:               "invalid access token",
			securitySchemeName: "AccessTokenUserBearer",
			setupRequest: func(r *http.Request) {
				r.Header.Set(token.AuthorizationHeader, "Bearer invalid-token-12345")
			},
			wantError:     true,
			wantErrorCode: apiError.InvalidAccessToken,
		},
		{
			name:               "cookie auth with valid CSRF for admin endpoint",
			securitySchemeName: "AccessTokenAdminBearer",
			setupRequest: func(r *http.Request) {
				accessToken := createAccessToken(t, role.RoleAdmin)
				csrfToken, _ := token.NewCSRFToken()

				// Set access token cookie
				r.AddCookie(&http.Cookie{
					Name:  token.AccessTokenName(),
					Value: accessToken,
				})

				// Set matching CSRF token in cookie and header
				r.AddCookie(&http.Cookie{
					Name:  token.CSRFTokenName(),
					Value: csrfToken,
				})
				r.Header.Set(token.CSRFTokenHeader, csrfToken)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			// Setup context with environment
			ctx := context.Background()
			e := env.New(map[string]string{
				"APP_SECRET":         appSecret,
				"APP_SECRET_VERSION": "1",
			})
			e.Logger = log.NullLogger()
			ctx = env.WithCtx(ctx, e)
			ctx = requestid.InjectRequestID(ctx, 12345)

			// Create authentication input
			input := &openapi3filter.AuthenticationInput{
				SecuritySchemeName: tt.securitySchemeName,
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request: req,
				},
			}

			// Call the middleware function
			err := OAPIAuthFunc(ctx, input)

			// Check error expectations
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				// Check error code if specified
				if tt.wantErrorCode != "" {
					var apiErr *apiError.Error
					if !errors.As(err, &apiErr) {
						t.Errorf("expected apiError, got %T", err)
						return
					}
					if apiErr.Code != tt.wantErrorCode {
						t.Errorf("expected error code %s, got %s", tt.wantErrorCode, apiErr.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestOAPIAuthFunc_ContextInjection(t *testing.T) {
	appSecret := "test-secret-32-bytes-long-12345"
	userID := int64(456)

	// Create a valid access token
	params := mJwt.JWTParams{
		UserID: fmt.Sprintf("%d", userID),
		Role:   role.RoleUser,
	}
	e := env.New(map[string]string{
		"APP_SECRET":         appSecret,
		"APP_SECRET_VERSION": "1",
	})
	accessToken, err := token.NewAccessToken(params, e)
	if err != nil {
		t.Fatalf("failed to create access token: %v", err)
	}

	// Create test request with Bearer token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(token.AuthorizationHeader, "Bearer "+accessToken)

	// Setup context
	ctx := context.Background()
	e.Logger = log.NullLogger()
	ctx = env.WithCtx(ctx, e)
	ctx = requestid.InjectRequestID(ctx, 12345)

	// Create authentication input
	input := &openapi3filter.AuthenticationInput{
		SecuritySchemeName: "AccessTokenUserBearer",
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: req,
		},
	}

	// Call the middleware function
	err = OAPIAuthFunc(ctx, input)
	if err != nil {
		t.Fatalf("OAPIAuthFunc() error = %v", err)
	}

	// Verify user ID was injected into request context
	updatedReq := input.RequestValidationInput.Request
	extractedUserID, err := token.UserIDFromCtx(updatedReq.Context())
	if err != nil {
		t.Errorf("expected user ID in context, got error: %v", err)
	}
	if extractedUserID != userID {
		t.Errorf("expected user ID %d, got %d", userID, extractedUserID)
	}

	// Verify access token was injected into request context
	extractedToken, err := token.AccessTokenFromCtx(updatedReq.Context())
	if err != nil {
		t.Errorf("expected access token in context, got error: %v", err)
	}
	if extractedToken == nil {
		t.Error("expected non-nil access token in context")
	}
}
