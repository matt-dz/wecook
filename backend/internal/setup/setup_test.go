package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestAppSecret_EnvironmentVariableSet(t *testing.T) {
	// Setup
	const secret = "secret"
	secretPath := filepath.Join(t.TempDir(), "secret")
	t.Setenv("APP_SECRET_PATH", secretPath)
	t.Setenv("APP_SECRET", secret)
	env := env.New(nil)
	if env.Get("APP_SECRET_PATH") != secretPath {
		t.Fatalf("APP_SECRET_PATH = %q, want %q", env.Get("APP_SECRET_PATH"), secretPath)
	}
	if env.Get("APP_SECRET") != secret {
		t.Fatalf("failed to set APP_SECRET = %q, want %q", env.Get("APP_SECRET"), secret)
	}

	err := AppSecret(env)
	if err != nil {
		t.Fatalf("AppSecret() received error: %v", err)
	}

	// Secret file should NOT exist
	_, err = os.Lstat(secretPath)
	if err == nil {
		t.Errorf("secret should not be saved - written to %q", secretPath)
	}

	// Expect secret
	if env.Get("APP_SECRET") != secret {
		t.Errorf("APP_SECRET = %q, want %q", env.Get("APP_SECRET"), secret)
	}
}

func TestAppSecret_NoEnvironmentVariable(t *testing.T) {
	// Setup
	const secretFileName = "secret"
	secretPath := filepath.Join(t.TempDir(), secretFileName)
	t.Setenv("APP_SECRET_PATH", secretPath)
	env := env.New(nil)
	if env.Get("APP_SECRET_PATH") != secretPath {
		t.Fatalf("APP_SECRET_PATH = %q, want %q", env.Get("APP_SECRET_PATH"), secretPath)
	}

	err := AppSecret(env)
	if err != nil {
		t.Fatalf("received error: %v", err)
	}

	// Secret file should exist
	f, err := os.Lstat(secretPath)
	if err != nil {
		t.Fatalf("secret file not written")
	}
	if f.Name() != secretFileName {
		t.Fatalf("secret file name = %q, want %q", f.Name(), secretFileName)
	}

	data, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("unexpected error when reading secret file: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("secret file is empty")
	}
	if env.Get("APP_SECRET") == "" {
		t.Errorf("APP_SECRET is empty")
	}
}

func TestAdmin(t *testing.T) {
	validPassword := "SecureP@ssw0rd123!"

	tests := []struct {
		name          string
		setup         func(*env.Env, *database.MockQuerier)
		wantError     bool
		wantErrorType error
	}{
		{
			name: "admin already exists - skip setup",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(1), nil)
			},
			wantError: false,
		},
		{
			name: "ADMIN_EMAIL not set - skip setup",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: false,
		},
		{
			name: "ADMIN_PASSWORD not set - skip setup",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: false,
		},
		{
			name: "ADMIN_FIRST_NAME missing - error",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError:     true,
			wantErrorType: &EnvironmentVariableMissingError{},
		},
		{
			name: "ADMIN_LAST_NAME missing - error",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError:     true,
			wantErrorType: &EnvironmentVariableMissingError{},
		},
		{
			name: "invalid email format - error",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "invalid-email")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: true,
		},
		{
			name: "weak password - too short",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", "Short1!")
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: true,
		},
		{
			name: "weak password - no uppercase",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", "password123!")
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: true,
		},
		{
			name: "weak password - no special character",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", "Password1234")
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), nil)
			},
			wantError: true,
		},
		{
			name: "database error on GetAdminCount - error",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

				mockDB.EXPECT().
					GetAdminCount(gomock.Any()).
					Return(int64(0), errors.New("database error"))
			},
			wantError: true,
		},
		{
			name: "database error on CreateAdmin - error",
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Admin")
				e.Set("ADMIN_LAST_NAME", "User")

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
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "admin@example.com")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "John")
				e.Set("ADMIN_LAST_NAME", "Doe")

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
			setup: func(e *env.Env, mockDB *database.MockQuerier) {
				e.Set("ADMIN_EMAIL", "  ADMIN@EXAMPLE.COM  ")
				e.Set("ADMIN_PASSWORD", validPassword)
				e.Set("ADMIN_FIRST_NAME", "Jane")
				e.Set("ADMIN_LAST_NAME", "Smith")

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

			tt.setup(e, mockDB)

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
