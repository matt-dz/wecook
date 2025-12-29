package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	mJwt "github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/log"
	"github.com/matt-dz/wecook/internal/role"
)

func TestPostApiLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	testPassword := "TestP@ssw0rd123!"
	passwordHash, err := argon2id.EncodeHash(testPassword, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to encode password: %v", err)
	}

	tests := []struct {
		name       string
		request    PostApiLoginRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful login",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "user@example.com",
					Password: testPassword,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "user@example.com").
					Return(database.GetUserRow{
						ID:           1,
						PasswordHash: passwordHash,
						Role:         database.RoleUser,
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantStatus: 200,
			wantCode:   "",
			wantError:  false,
		},
		{
			name: "user not found",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "nonexistent@example.com",
					Password: testPassword,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "nonexistent@example.com").
					Return(database.GetUserRow{}, pgx.ErrNoRows)
			},
			wantStatus: 401,
			wantCode:   apiError.InvalidCredentials.String(),
			wantError:  false,
		},
		{
			name: "incorrect password",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "user@example.com",
					Password: "WrongP@ssw0rd123!",
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "user@example.com").
					Return(database.GetUserRow{
						ID:           1,
						PasswordHash: passwordHash,
						Role:         database.RoleUser,
					}, nil)
			},
			wantStatus: 401,
			wantCode:   apiError.InvalidCredentials.String(),
			wantError:  false,
		},
		{
			name: "database error on GetUser",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "user@example.com",
					Password: testPassword,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "user@example.com").
					Return(database.GetUserRow{}, errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "malformed password hash",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "user@example.com",
					Password: testPassword,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "user@example.com").
					Return(database.GetUserRow{
						ID:           1,
						PasswordHash: "invalid-hash-format",
						Role:         database.RoleUser,
					}, nil)
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on UpdateUserRefreshTokenHash",
			request: PostApiLoginRequestObject{
				Body: &UserLoginRequest{
					Email:    "user@example.com",
					Password: testPassword,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUser(gomock.Any(), "user@example.com").
					Return(database.GetUserRow{
						ID:           1,
						PasswordHash: passwordHash,
						Role:         database.RoleUser,
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			e := env.New(map[string]string{
				"APP_SECRET": "test-secret-key-for-jwt-signing",
			})
			e.Logger = log.NullLogger()
			e.Database = mockDB
			ctx = env.WithCtx(ctx, e)

			resp, err := server.PostApiLogin(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiLogin() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case loginSuccessResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if v.accessCookie == nil {
					t.Error("expected access cookie, got nil")
				}
				if v.refreshCookie == nil {
					t.Error("expected refresh cookie, got nil")
				}
				if v.csrfCookie == nil {
					t.Error("expected CSRF cookie, got nil")
				}
				if v.body.AccessToken == "" {
					t.Error("expected access token in body, got empty string")
				}
			case PostApiLogin401JSONResponse:
				if tt.wantStatus != 401 {
					t.Errorf("expected status %d, got 401", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiLogin500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestPostApiLogin_AdminRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	testPassword := "AdminP@ssw0rd123!"
	passwordHash, err := argon2id.EncodeHash(testPassword, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to encode password: %v", err)
	}

	mockDB.EXPECT().
		GetUser(gomock.Any(), "admin@example.com").
		Return(database.GetUserRow{
			ID:           1,
			PasswordHash: passwordHash,
			Role:         database.RoleAdmin,
		}, nil)

	mockDB.EXPECT().
		UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, params database.UpdateUserRefreshTokenHashParams) error {
			if params.ID != 1 {
				t.Errorf("expected user ID 1, got %d", params.ID)
			}
			if !params.RefreshTokenHash.Valid {
				t.Error("expected valid refresh token hash")
			}
			if params.RefreshTokenHash.String == "" {
				t.Error("expected non-empty refresh token hash")
			}
			return nil
		})

	ctx := context.Background()
	ctx = requestid.InjectRequestID(ctx, 12345)
	e := env.New(
		map[string]string{
			"APP_SECRET": "test-secret-key-for-jwt-signing",
		},
	)
	e.Logger = log.NullLogger()
	e.Database = mockDB
	ctx = env.WithCtx(ctx, e)

	request := PostApiLoginRequestObject{
		Body: &UserLoginRequest{
			Email:    "admin@example.com",
			Password: testPassword,
		},
	}

	resp, err := server.PostApiLogin(ctx, request)
	if err != nil {
		t.Fatalf("PostApiLogin() error = %v", err)
	}

	successResp, ok := resp.(loginSuccessResponse)
	if !ok {
		t.Fatalf("expected loginSuccessResponse, got %T", resp)
	}

	if successResp.accessCookie == nil {
		t.Error("expected access cookie, got nil")
	}
	if successResp.refreshCookie == nil {
		t.Error("expected refresh cookie, got nil")
	}
	if successResp.csrfCookie == nil {
		t.Error("expected CSRF cookie, got nil")
	}
	if successResp.body.AccessToken == "" {
		t.Error("expected access token in body, got empty string")
	}
	if successResp.body.TokenType == nil || *successResp.body.TokenType != "Bearer" {
		t.Error("expected token type 'Bearer'")
	}
	if successResp.body.ExpiresIn == nil {
		t.Error("expected expiresIn to be set")
	}
}

func TestPostApiAuthRefresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	userID := int64(123)
	validRefreshToken, err := token.NewRefreshToken(userID)
	if err != nil {
		t.Fatalf("failed to create test refresh token: %v", err)
	}

	validRefreshTokenHash, err := argon2id.EncodeHash(validRefreshToken, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to hash test refresh token: %v", err)
	}

	newRefreshToken, err := token.NewRefreshToken(userID)
	if err != nil {
		t.Fatalf("failed to create new test refresh token: %v", err)
	}

	tests := []struct {
		name       string
		request    PostApiAuthRefreshRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful token refresh via body",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(24 * time.Hour),
							Valid: true,
						},
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					GetUserRole(gomock.Any(), userID).
					Return(database.RoleUser, nil)
			},
			wantStatus: 200,
			wantCode:   "",
			wantError:  false,
		},
		{
			name: "successful token refresh via query param",
			request: PostApiAuthRefreshRequestObject{
				Params: PostApiAuthRefreshParams{
					Refresh: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(24 * time.Hour),
							Valid: true,
						},
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					GetUserRole(gomock.Any(), userID).
					Return(database.RoleUser, nil)
			},
			wantStatus: 200,
			wantCode:   "",
			wantError:  false,
		},
		{
			name: "missing refresh token",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{},
			},
			setup:      func() {},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "malformed refresh token - no dot separator",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: stringPtr("invalidtoken"),
				},
			},
			setup:      func() {},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "malformed refresh token - invalid user ID",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: stringPtr("notanumber.randomdata"),
				},
			},
			setup:      func() {},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "user not found",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{}, pgx.ErrNoRows)
			},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "database error on get refresh token",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "refresh token mismatch",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &newRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(24 * time.Hour),
							Valid: true,
						},
					}, nil)
			},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "expired refresh token",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(-24 * time.Hour),
							Valid: true,
						},
					}, nil)
			},
			wantStatus: 401,
			wantCode:   apiError.InvalidRefreshToken.String(),
			wantError:  false,
		},
		{
			name: "database error on update refresh token",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(24 * time.Hour),
							Valid: true,
						},
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on get user role",
			request: PostApiAuthRefreshRequestObject{
				Body: &PostApiAuthRefreshJSONRequestBody{
					RefreshToken: &validRefreshToken,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUserRefreshTokenHash(gomock.Any(), userID).
					Return(database.GetUserRefreshTokenHashRow{
						RefreshTokenHash: pgtype.Text{
							String: validRefreshTokenHash,
							Valid:  true,
						},
						RefreshTokenExpiresAt: pgtype.Timestamptz{
							Time:  time.Now().Add(24 * time.Hour),
							Valid: true,
						},
					}, nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(nil)

				mockDB.EXPECT().
					GetUserRole(gomock.Any(), userID).
					Return(database.Role(""), errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			e := env.New(
				map[string]string{
					"APP_SECRET": "test-secret-key-for-jwt-signing",
				},
			)
			e.Logger = log.NullLogger()
			e.Database = mockDB
			ctx = env.WithCtx(ctx, e)

			resp, err := server.PostApiAuthRefresh(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiAuthRefresh() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case loginSuccessResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if v.body.AccessToken == "" {
					t.Error("expected access token, got empty string")
				}
				if v.accessCookie == nil {
					t.Error("expected access cookie, got nil")
				}
				if v.refreshCookie == nil {
					t.Error("expected refresh cookie, got nil")
				}
				if v.csrfCookie == nil {
					t.Error("expected CSRF cookie, got nil")
				}
			case PostApiAuthRefresh401JSONResponse:
				if tt.wantStatus != 401 {
					t.Errorf("expected status %d, got 401", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiAuthRefresh500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestGetApiAuthVerify(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name       string
		userRole   string
		queryRole  *Role
		setup      func() context.Context
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name:     "successful verification - user role with no query param",
			userRole: "user",
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))

				accessToken, err := token.NewAccessToken(mJwt.JWTParams{
					Role:   role.RoleUser,
					UserID: "123",
				}, env.EnvFromCtx(ctx))
				if err != nil {
					t.Fatalf("failed to create access token: %v", err)
				}

				parsedToken, err := mJwt.ValidateJWT(accessToken, "1", []byte("test-secret-key-for-jwt-signing"))
				if err != nil {
					t.Fatalf("failed to validate token: %v", err)
				}

				ctx = token.AccessTokenWithCtx(ctx, parsedToken)
				return ctx
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name:      "successful verification - user role with explicit user query param",
			userRole:  "user",
			queryRole: rolePtr(RoleUser),
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))

				accessToken, err := token.NewAccessToken(mJwt.JWTParams{
					Role:   role.RoleUser,
					UserID: "123",
				}, env.EnvFromCtx(ctx))
				if err != nil {
					t.Fatalf("failed to create access token: %v", err)
				}

				parsedToken, err := mJwt.ValidateJWT(accessToken, "1", []byte("test-secret-key-for-jwt-signing"))
				if err != nil {
					t.Fatalf("failed to validate token: %v", err)
				}

				ctx = token.AccessTokenWithCtx(ctx, parsedToken)
				return ctx
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name:      "successful verification - admin role checking user permission",
			userRole:  "admin",
			queryRole: rolePtr(RoleUser),
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))

				accessToken, err := token.NewAccessToken(mJwt.JWTParams{
					Role:   role.RoleAdmin,
					UserID: "456",
				}, env.EnvFromCtx(ctx))
				if err != nil {
					t.Fatalf("failed to create access token: %v", err)
				}

				parsedToken, err := mJwt.ValidateJWT(accessToken, "1", []byte("test-secret-key-for-jwt-signing"))
				if err != nil {
					t.Fatalf("failed to validate token: %v", err)
				}

				ctx = token.AccessTokenWithCtx(ctx, parsedToken)
				return ctx
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name:      "successful verification - admin role checking admin permission",
			userRole:  "admin",
			queryRole: rolePtr(RoleAdmin),
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))

				accessToken, err := token.NewAccessToken(mJwt.JWTParams{
					Role:   role.RoleAdmin,
					UserID: "456",
				}, env.EnvFromCtx(ctx))
				if err != nil {
					t.Fatalf("failed to create access token: %v", err)
				}

				parsedToken, err := mJwt.ValidateJWT(accessToken, "1", []byte("test-secret-key-for-jwt-signing"))
				if err != nil {
					t.Fatalf("failed to validate token: %v", err)
				}

				ctx = token.AccessTokenWithCtx(ctx, parsedToken)
				return ctx
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name:      "insufficient permissions - user role trying to check admin permission",
			userRole:  "user",
			queryRole: rolePtr(RoleAdmin),
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))

				accessToken, err := token.NewAccessToken(mJwt.JWTParams{
					Role:   role.RoleUser,
					UserID: "123",
				}, env.EnvFromCtx(ctx))
				if err != nil {
					t.Fatalf("failed to create access token: %v", err)
				}

				parsedToken, err := mJwt.ValidateJWT(accessToken, "1", []byte("test-secret-key-for-jwt-signing"))
				if err != nil {
					t.Fatalf("failed to validate token: %v", err)
				}

				ctx = token.AccessTokenWithCtx(ctx, parsedToken)
				return ctx
			},
			wantStatus: 401,
			wantCode:   apiError.InsufficientPermissions.String(),
			wantError:  false,
		},
		{
			name:     "missing access token in context",
			userRole: "",
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, env.New(
					map[string]string{
						"APP_SECRET": "test-secret-key-for-jwt-signing",
					},
				))
				return ctx
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()

			request := GetApiAuthVerifyRequestObject{
				Params: GetApiAuthVerifyParams{
					Role: tt.queryRole,
				},
			}

			resp, err := server.GetApiAuthVerify(ctx, request)
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiAuthVerify() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case GetApiAuthVerify204Response:
				if tt.wantStatus != 204 {
					t.Errorf("expected status %d, got 204", tt.wantStatus)
				}
			case GetApiAuthVerify401JSONResponse:
				if tt.wantStatus != 401 {
					t.Errorf("expected status %d, got 401", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case GetApiAuthVerify500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func rolePtr(r Role) *Role {
	return &r
}

func stringPtr(s string) *string {
	return &s
}
