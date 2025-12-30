package setup

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/matt-dz/wecook/internal/config"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestAdmin(t *testing.T) {
	validPassword := config.AdminPassword("SecureP@ssw0rd123!")

	tests := []struct {
		name          string
		setup         func(*config.Config, *database.MockQuerier)
		wantError     bool
		wantErrorType error
	}{
		{
			name: "admin already exists - skip setup",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "admin@example.com"
				c.Admin.Password = validPassword
				c.Admin.FirstName = "Admin"
				c.Admin.LastName = "User"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(1), nil)
			},
			wantError: false,
		},
		{
			name: "ADMIN_EMAIL not set - skip setup",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Password = validPassword
				c.Admin.FirstName = "Admin"
				c.Admin.LastName = "User"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: false,
		},
		{
			name: "ADMIN_PASSWORD not set - skip setup",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "admin@example.com"
				c.Admin.FirstName = "Admin"
				c.Admin.LastName = "User"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: false,
		},
		{
			name: "database error on GetAdminCount - error",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "admin@example.com"
				c.Admin.Password = validPassword
				c.Admin.FirstName = "Admin"
				c.Admin.LastName = "User"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), errors.New("database error"))
			},
			wantError: true,
		},
		{
			name: "database error on CreateAdmin - error",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "admin@example.com"
				c.Admin.Password = validPassword
				c.Admin.FirstName = "Admin"
				c.Admin.LastName = "User"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)

				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("create admin error"))
			},
			wantError: true,
		},
		{
			name: "successful admin creation",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "admin@example.com"
				c.Admin.Password = validPassword
				c.Admin.FirstName = "John"
				c.Admin.LastName = "Doe"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)

				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, params database.CreateAdminParams) (int64, error) {
						// Verify the correct parameters were passed
						if params.FirstName != "John" {
							t.Errorf("expected FirstName 'John', got %q", params.FirstName)
						}
						if params.LastName != "Doe" {
							t.Errorf("expected LastName 'Doe', got %q", params.LastName)
						}
						if params.Email != "admin@example.com" {
							t.Errorf("expected Email 'admin@example.com', got %q", params.Email)
						}
						// Verify password hash is not empty
						if params.PasswordHash == "" {
							t.Error("password hash should not be empty")
						}
						return int64(1), nil
					})
			},
			wantError: false,
		},
		{
			name: "successful admin creation with email normalization",
			setup: func(c *config.Config, mockDB *database.MockQuerier) {
				c.Admin.Email = "  ADMIN@EXAMPLE.COM  "
				c.Admin.Password = validPassword
				c.Admin.FirstName = "Jane"
				c.Admin.LastName = "Smith"

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)

				mockDB.EXPECT().
					CreateAdmin(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, params database.CreateAdminParams) (int64, error) {
						if params.Email != "  ADMIN@EXAMPLE.COM  " {
							t.Errorf("expected email to be passed as-is to database (database will normalize), got %q", params.Email)
						}
						return int64(1), nil
					})
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := database.NewMockQuerier(ctrl)
			e := env.New(nil)
			e.Logger = log.NullLogger()
			e.Database = mockDB

			tt.setup(&e.Config, mockDB)

			ctx := context.Background()
			err := Admin(ctx, e)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				if tt.wantErrorType != nil {
					var envErr *EnvironmentVariableMissingError
					if !errors.As(err, &envErr) {
						t.Errorf("expected error type %T, got %T: %v", &EnvironmentVariableMissingError{}, err, err)
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
