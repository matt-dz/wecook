package client

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestPostApiAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PostApiAdminRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful admin creation",
			request: PostApiAdminRequestObject{
				Body: &CreateAdminRequest{
					Email:     openapi_types.Email("admin@example.com"),
					FirstName: "John",
					LastName:  "Doe",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(1), nil)
			},
			wantStatus: 204,
			wantCode:   "",
			wantError:  false,
		},
		{
			name: "weak password",
			request: PostApiAdminRequestObject{
				Body: &CreateAdminRequest{
					Email:     openapi_types.Email("admin@example.com"),
					FirstName: "John",
					LastName:  "Doe",
					Password:  "weak",
				},
			},
			setup:      func() {},
			wantStatus: 422,
			wantCode:   apiError.WeakPassword.String(),
			wantError:  false,
		},
		{
			name: "duplicate email",
			request: PostApiAdminRequestObject{
				Body: &CreateAdminRequest{
					Email:     openapi_types.Email("admin@example.com"),
					FirstName: "John",
					LastName:  "Doe",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "users_unique_email",
				}
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(0), pgErr)
			},
			wantStatus: 409,
			wantCode:   apiError.WeakPassword.String(),
			wantError:  false,
		},
		{
			name: "database error",
			request: PostApiAdminRequestObject{
				Body: &CreateAdminRequest{
					Email:     openapi_types.Email("admin@example.com"),
					FirstName: "John",
					LastName:  "Doe",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database error"))
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
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			resp, err := server.PostApiAdmin(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiAdmin() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PostApiAdmin204JSONResponse:
				if tt.wantStatus != 204 {
					t.Errorf("expected status %d, got 204", tt.wantStatus)
				}
			case PostApiAdmin422JSONResponse:
				if tt.wantStatus != 422 {
					t.Errorf("expected status %d, got 422", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiAdmin409JSONResponse:
				if tt.wantStatus != 409 {
					t.Errorf("expected status %d, got 409", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiAdmin500JSONResponse:
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

func TestPostApiAdminUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PostApiAdminUserRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful user creation",
			request: PostApiAdminUserRequestObject{
				Body: &CreateUserRequest{
					Email:     openapi_types.Email("user@example.com"),
					FirstName: "Jane",
					LastName:  "Smith",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(2), nil)
			},
			wantStatus: 204,
			wantCode:   "",
			wantError:  false,
		},
		{
			name: "weak password",
			request: PostApiAdminUserRequestObject{
				Body: &CreateUserRequest{
					Email:     openapi_types.Email("user@example.com"),
					FirstName: "Jane",
					LastName:  "Smith",
					Password:  "short",
				},
			},
			setup:      func() {},
			wantStatus: 409,
			wantCode:   apiError.WeakPassword.String(),
			wantError:  false,
		},
		{
			name: "duplicate email",
			request: PostApiAdminUserRequestObject{
				Body: &CreateUserRequest{
					Email:     openapi_types.Email("user@example.com"),
					FirstName: "Jane",
					LastName:  "Smith",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				pgErr := &pgconn.PgError{
					Code:           "23505",
					ConstraintName: "users_unique_email",
				}
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(0), pgErr)
			},
			wantStatus: 422,
			wantCode:   apiError.EmailConflict.String(),
			wantError:  false,
		},
		{
			name: "database error",
			request: PostApiAdminUserRequestObject{
				Body: &CreateUserRequest{
					Email:     openapi_types.Email("user@example.com"),
					FirstName: "Jane",
					LastName:  "Smith",
					Password:  "StrongP@ssw0rd123!",
				},
			},
			setup: func() {
				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("connection failed"))
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
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			resp, err := server.PostApiAdminUser(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiAdminUser() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PostApiAdminUser204JSONResponse:
				if tt.wantStatus != 204 {
					t.Errorf("expected status %d, got 204", tt.wantStatus)
				}
			case PostApiAdminUser409JSONResponse:
				if tt.wantStatus != 409 {
					t.Errorf("expected status %d, got 409", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiAdminUser422JSONResponse:
				if tt.wantStatus != 422 {
					t.Errorf("expected status %d, got 422", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiAdminUser500JSONResponse:
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
