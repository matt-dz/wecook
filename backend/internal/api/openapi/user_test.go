package client

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/dbmock"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestPostApiLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmock.NewMockQuerier(ctrl)
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
			ctx = env.WithCtx(ctx, env.New(
				log.NullLogger(),
				&database.Database{
					Querier: mockDB,
				},
				nil,
				nil,
				map[string]string{
					"APP_SECRET": "test-secret-key-for-jwt-signing",
				},
			))

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

	mockDB := dbmock.NewMockQuerier(ctrl)
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
	ctx = env.WithCtx(ctx, env.New(
		log.NullLogger(),
		&database.Database{
			Querier: mockDB,
		},
		nil,
		nil,
		map[string]string{
			"APP_SECRET": "test-secret-key-for-jwt-signing",
		},
	))

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
