package client

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/config"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestGetApiPreferences(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name             string
		setup            func()
		wantStatus       int
		wantCode         string
		wantPublicSignup bool
		wantError        bool
	}{
		{
			name: "successful retrieval - allow public signup true",
			setup: func() {
				mockDB.EXPECT().
					GetPreferences(gomock.Any(), int32(config.PreferenceID)).
					Return(database.Preference{
						ID:                config.PreferenceID,
						AllowPublicSignup: true,
					}, nil)
			},
			wantStatus:       200,
			wantPublicSignup: true,
			wantError:        false,
		},
		{
			name: "successful retrieval - allow public signup false",
			setup: func() {
				mockDB.EXPECT().
					GetPreferences(gomock.Any(), int32(config.PreferenceID)).
					Return(database.Preference{
						ID:                config.PreferenceID,
						AllowPublicSignup: false,
					}, nil)
			},
			wantStatus:       200,
			wantPublicSignup: false,
			wantError:        false,
		},
		{
			name: "database error",
			setup: func() {
				mockDB.EXPECT().
					GetPreferences(gomock.Any(), int32(config.PreferenceID)).
					Return(database.Preference{}, errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			e := env.New(nil)
			e.Logger = log.NullLogger()
			e.Database = mockDB

			ctx := context.Background()
			ctx = env.WithCtx(ctx, e)
			ctx = requestid.InjectRequestID(ctx, 12345)

			request := GetApiPreferencesRequestObject{}
			response, err := server.GetApiPreferences(ctx, request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				resp, ok := response.(GetApiPreferences500JSONResponse)
				if !ok {
					t.Fatalf("expected GetApiPreferences500JSONResponse, got %T", response)
				}
				if resp.Status != tt.wantStatus {
					t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Status)
				}
				if resp.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, resp.Code)
				}
			} else {
				resp, ok := response.(GetApiPreferences200JSONResponse)
				if !ok {
					t.Fatalf("expected GetApiPreferences200JSONResponse, got %T", response)
				}
				if resp.AllowPublicSignup != tt.wantPublicSignup {
					t.Errorf("expected AllowPublicSignup %v, got %v", tt.wantPublicSignup, resp.AllowPublicSignup)
				}
			}
		})
	}
}

func TestPatchApiPreferences(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	boolPtr := func(b bool) *bool {
		return &b
	}

	tests := []struct {
		name              string
		allowPublicSignup *bool
		setup             func(*bool)
		wantStatus        int
		wantCode          string
		wantPublicSignup  bool
		wantError         bool
	}{
		{
			name:              "successful update - set to true",
			allowPublicSignup: boolPtr(true),
			setup: func(val *bool) {
				mockDB.EXPECT().
					UpdatePreferences(gomock.Any(), database.UpdatePreferencesParams{
						ID: config.PreferenceID,
						UpdateAllowPublicSignup: pgtype.Bool{
							Bool:  true,
							Valid: true,
						},
						AllowPublicSignup: pgtype.Bool{
							Bool:  *val,
							Valid: true,
						},
					}).
					Return(database.Preference{
						ID:                config.PreferenceID,
						AllowPublicSignup: *val,
					}, nil)
			},
			wantStatus:       200,
			wantPublicSignup: true,
			wantError:        false,
		},
		{
			name:              "successful update - set to false",
			allowPublicSignup: boolPtr(false),
			setup: func(val *bool) {
				mockDB.EXPECT().
					UpdatePreferences(gomock.Any(), database.UpdatePreferencesParams{
						ID: config.PreferenceID,
						UpdateAllowPublicSignup: pgtype.Bool{
							Bool:  true,
							Valid: true,
						},
						AllowPublicSignup: pgtype.Bool{
							Bool:  *val,
							Valid: true,
						},
					}).
					Return(database.Preference{
						ID:                config.PreferenceID,
						AllowPublicSignup: *val,
					}, nil)
			},
			wantStatus:       200,
			wantPublicSignup: false,
			wantError:        false,
		},
		{
			name:              "successful update - nil value (no update field set)",
			allowPublicSignup: nil,
			setup: func(val *bool) {
				mockDB.EXPECT().
					UpdatePreferences(gomock.Any(), database.UpdatePreferencesParams{
						ID: config.PreferenceID,
					}).
					Return(database.Preference{
						ID:                config.PreferenceID,
						AllowPublicSignup: true, // existing value unchanged
					}, nil)
			},
			wantStatus:       200,
			wantPublicSignup: true,
			wantError:        false,
		},
		{
			name:              "database error",
			allowPublicSignup: boolPtr(true),
			setup: func(val *bool) {
				mockDB.EXPECT().
					UpdatePreferences(gomock.Any(), gomock.Any()).
					Return(database.Preference{}, errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.allowPublicSignup)

			e := env.New(nil)
			e.Logger = log.NullLogger()
			e.Database = mockDB

			ctx := context.Background()
			ctx = env.WithCtx(ctx, e)
			ctx = requestid.InjectRequestID(ctx, 12345)

			request := PatchApiPreferencesRequestObject{
				Body: &PatchApiPreferencesJSONRequestBody{
					AllowPublicSignup: tt.allowPublicSignup,
				},
			}
			response, err := server.PatchApiPreferences(ctx, request)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				resp, ok := response.(PatchApiPreferences500JSONResponse)
				if !ok {
					t.Fatalf("expected PatchApiPreferences500JSONResponse, got %T", response)
				}
				if resp.Status != tt.wantStatus {
					t.Errorf("expected status %d, got %d", tt.wantStatus, resp.Status)
				}
				if resp.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, resp.Code)
				}
			} else {
				resp, ok := response.(PatchApiPreferences200JSONResponse)
				if !ok {
					t.Fatalf("expected PatchApiPreferences200JSONResponse, got %T", response)
				}
				if resp.AllowPublicSignup != tt.wantPublicSignup {
					t.Errorf("expected AllowPublicSignup %v, got %v", tt.wantPublicSignup, resp.AllowPublicSignup)
				}
			}
		})
	}
}
