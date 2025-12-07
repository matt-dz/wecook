package client

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/database"
	dbmoc "github.com/matt-dz/wecook/internal/dbmock"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestPostApiRecipes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmoc.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PostApiRecipesRequestObject
		userID     int64
		injectUser bool
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
		wantID     int64
	}{
		{
			name:       "successful recipe creation",
			request:    PostApiRecipesRequestObject{},
			userID:     123,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CreateRecipe(gomock.Any(), gomock.Any()).
					Return(int64(456), nil)
			},
			wantStatus: 201,
			wantCode:   "",
			wantError:  false,
			wantID:     456,
		},
		{
			name:       "missing user id in context",
			request:    PostApiRecipesRequestObject{},
			injectUser: false,
			setup:      func() {},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			wantID:     0,
		},
		{
			name:       "database error",
			request:    PostApiRecipesRequestObject{},
			userID:     123,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CreateRecipe(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			wantID:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			if tt.injectUser {
				ctx = token.UserIDWithCtx(ctx, tt.userID)
			}
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			resp, err := server.PostApiRecipes(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiRecipes() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PostApiRecipes201JSONResponse:
				if tt.wantStatus != 201 {
					t.Errorf("expected status %d, got 201", tt.wantStatus)
				}
				if v.RecipeId != tt.wantID {
					t.Errorf("expected recipe ID %d, got %d", tt.wantID, v.RecipeId)
				}
			case PostApiRecipes500JSONResponse:
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
