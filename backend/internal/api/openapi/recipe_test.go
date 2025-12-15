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
	"github.com/matt-dz/wecook/internal/database"
	dbmoc "github.com/matt-dz/wecook/internal/dbmock"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/filestore"
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

func TestGetApiRecipesRecipeID(t *testing.T) {
	server := NewServer()

	now := time.Now()
	cookTimeUnit := database.TimeUnitMinutes
	prepTimeUnit := database.TimeUnitHours

	tests := []struct {
		name       string
		request    GetApiRecipesRecipeIDRequestObject
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject)
	}{
		{
			name: "successful recipe retrieval with all fields",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetPublishedRecipeAndOwnerRow{
						ID:             123,
						UserID:         pgtype.Int8{Int64: 456, Valid: true},
						Title:          "Test Recipe",
						Description:    pgtype.Text{String: "A delicious test recipe", Valid: true},
						ImageUrl:       pgtype.Text{String: "recipe.jpg", Valid: true},
						Published:      true,
						CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
						CookTimeAmount: pgtype.Int4{Int32: 30, Valid: true},
						CookTimeUnit:   database.NullTimeUnit{TimeUnit: cookTimeUnit, Valid: true},
						PrepTimeAmount: pgtype.Int4{Int32: 15, Valid: true},
						PrepTimeUnit:   database.NullTimeUnit{TimeUnit: prepTimeUnit, Valid: true},
						Servings:       pgtype.Float4{Float32: 4.0, Valid: true},
						FirstName:      "John",
						LastName:       "Doe",
						ID_2:           456,
					}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{
						{
							ID:          1,
							RecipeID:    123,
							StepNumber:  1,
							Instruction: pgtype.Text{String: "Mix ingredients", Valid: true},
							ImageUrl:    pgtype.Text{String: "step1.jpg", Valid: true},
						},
						{
							ID:          2,
							RecipeID:    123,
							StepNumber:  2,
							Instruction: pgtype.Text{String: "Bake for 30 minutes", Valid: true},
							ImageUrl:    pgtype.Text{String: "", Valid: false},
						},
					}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return([]database.RecipeIngredient{
						{
							ID:       1,
							RecipeID: 123,
							Name:     pgtype.Text{String: "Flour", Valid: true},
							Quantity: pgtype.Float4{Float32: 2.0, Valid: true},
							Unit:     pgtype.Text{String: "cups", Valid: true},
							ImageUrl: pgtype.Text{String: "flour.jpg", Valid: true},
						},
						{
							ID:       2,
							RecipeID: 123,
							Name:     pgtype.Text{String: "Sugar", Valid: true},
							Quantity: pgtype.Float4{Float32: 1.0, Valid: true},
							Unit:     pgtype.Text{String: "cup", Valid: true},
							ImageUrl: pgtype.Text{String: "", Valid: false},
						},
					}, nil)

				// Expect FileURL calls for recipe and step images
				mockFS.EXPECT().FileURL("recipe.jpg").Return("http://localhost:8080/recipe.jpg")
				mockFS.EXPECT().FileURL("step1.jpg").Return("http://localhost:8080/step1.jpg")
				mockFS.EXPECT().FileURL("flour.jpg").Return("http://localhost:8080/flour.jpg")
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID200JSONResponse, got %T", resp)
					return
				}
				if v.Recipe.Id != 123 {
					t.Errorf("expected recipe ID 123, got %d", v.Recipe.Id)
				}
				if v.Recipe.Title != "Test Recipe" {
					t.Errorf("expected title 'Test Recipe', got %s", v.Recipe.Title)
				}
				if v.Recipe.Description == nil || *v.Recipe.Description != "A delicious test recipe" {
					t.Errorf("expected description 'A delicious test recipe', got %v", v.Recipe.Description)
				}
				if v.Owner.FirstName != "John" {
					t.Errorf("expected owner first name 'John', got %s", v.Owner.FirstName)
				}
				if v.Owner.LastName != "Doe" {
					t.Errorf("expected owner last name 'Doe', got %s", v.Owner.LastName)
				}
				if len(v.Recipe.Steps) != 2 {
					t.Errorf("expected 2 steps, got %d", len(v.Recipe.Steps))
				}
				if len(v.Recipe.Ingredients) != 2 {
					t.Errorf("expected 2 ingredients, got %d", len(v.Recipe.Ingredients))
				}
				if v.Recipe.CookTimeAmount == nil || *v.Recipe.CookTimeAmount != 30 {
					t.Errorf("expected cook time amount 30, got %v", v.Recipe.CookTimeAmount)
				}
				if v.Recipe.Servings == nil || *v.Recipe.Servings != 4.0 {
					t.Errorf("expected servings 4.0, got %v", v.Recipe.Servings)
				}
			},
		},
		{
			name: "recipe not found",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 999,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(999)).
					Return(database.GetPublishedRecipeAndOwnerRow{}, pgx.ErrNoRows)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID404JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID404JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error on recipe retrieval",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetPublishedRecipeAndOwnerRow{}, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID500JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "database error on steps retrieval",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetPublishedRecipeAndOwnerRow{
						ID:        123,
						UserID:    pgtype.Int8{Int64: 456, Valid: true},
						Title:     "Test Recipe",
						Published: true,
						CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						FirstName: "John",
						LastName:  "Doe",
						ID_2:      456,
					}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return(nil, errors.New("failed to fetch steps"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID500JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ingredients retrieval",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetPublishedRecipeAndOwnerRow{
						ID:        123,
						UserID:    pgtype.Int8{Int64: 456, Valid: true},
						Title:     "Test Recipe",
						Published: true,
						CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						FirstName: "John",
						LastName:  "Doe",
						ID_2:      456,
					}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return(nil, errors.New("failed to fetch ingredients"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID500JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "recipe with empty steps and ingredients",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublishedRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetPublishedRecipeAndOwnerRow{
						ID:        123,
						UserID:    pgtype.Int8{Int64: 456, Valid: true},
						Title:     "Minimal Recipe",
						Published: true,
						CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						FirstName: "Jane",
						LastName:  "Smith",
						ID_2:      456,
					}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return([]database.RecipeIngredient{}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesRecipeID200JSONResponse, got %T", resp)
					return
				}
				if len(v.Recipe.Steps) != 0 {
					t.Errorf("expected 0 steps, got %d", len(v.Recipe.Steps))
				}
				if len(v.Recipe.Ingredients) != 0 {
					t.Errorf("expected 0 ingredients, got %d", len(v.Recipe.Ingredients))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbmoc.NewMockQuerier(ctrl)
			mockFS := filestore.NewMockFileStoreInterface(ctrl)

			tt.setup(mockDB, mockFS)

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
				FileStore: mockFS,
			})

			resp, err := server.GetApiRecipesRecipeID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiRecipesRecipeID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestDeleteApiRecipesRecipeID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmoc.NewMockQuerier(ctrl)
	mockFS := filestore.NewMockFileStoreInterface(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    DeleteApiRecipesRecipeIDRequestObject
		userID     int64
		injectUser bool
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful deletion",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return([]database.RecipeIngredient{}, nil)

				mockDB.EXPECT().
					DeleteRecipe(gomock.Any(), int64(123)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name: "successful deletion with images",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{
						ImageUrl: pgtype.Text{String: "files/covers/123.jpg", Valid: true},
					}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{
						{
							ID:       1,
							RecipeID: 123,
							ImageUrl: pgtype.Text{String: "files/steps/123/1.jpg", Valid: true},
						},
						{
							ID:       2,
							RecipeID: 123,
							ImageUrl: pgtype.Text{String: "files/steps/123/2.jpg", Valid: true},
						},
					}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return([]database.RecipeIngredient{
						{
							ID:       1,
							RecipeID: 123,
							ImageUrl: pgtype.Text{String: "files/ingredients/123/1.jpg", Valid: true},
						},
					}, nil)

				// Expect Delete calls for all images
				mockFS.EXPECT().DeleteURLPath("files/covers/123.jpg").Return(nil)
				mockFS.EXPECT().DeleteURLPath("files/steps/123/1.jpg").Return(nil)
				mockFS.EXPECT().DeleteURLPath("files/steps/123/2.jpg").Return(nil)
				mockFS.EXPECT().DeleteURLPath("files/ingredients/123/1.jpg").Return(nil)

				mockDB.EXPECT().
					DeleteRecipe(gomock.Any(), int64(123)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name: "missing user id in context",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			injectUser: false,
			setup:      func() {},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on ownership check",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(false, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "user does not own recipe",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
		},
		{
			name: "database error on get recipe",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on get steps",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return(nil, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on get ingredients",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return(nil, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error on deletion",
			request: DeleteApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, nil)

				mockDB.EXPECT().
					GetRecipeSteps(gomock.Any(), int64(123)).
					Return([]database.RecipeStep{}, nil)

				mockDB.EXPECT().
					GetRecipeIngredients(gomock.Any(), int64(123)).
					Return([]database.RecipeIngredient{}, nil)

				mockDB.EXPECT().
					DeleteRecipe(gomock.Any(), int64(123)).
					Return(errors.New("failed to delete recipe"))
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
			if tt.injectUser {
				ctx = token.UserIDWithCtx(ctx, tt.userID)
			}
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
				FileStore: mockFS,
			})

			resp, err := server.DeleteApiRecipesRecipeID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteApiRecipesRecipeID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case DeleteApiRecipesRecipeID204Response:
				if tt.wantStatus != 204 {
					t.Errorf("expected status %d, got 204", tt.wantStatus)
				}
			case DeleteApiRecipesRecipeID404JSONResponse:
				if tt.wantStatus != 404 {
					t.Errorf("expected status %d, got 404", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case DeleteApiRecipesRecipeID500JSONResponse:
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

func TestPostApiRecipesRecipeIDIngredients(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmoc.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PostApiRecipesRecipeIDIngredientsRequestObject
		userID     int64
		injectUser bool
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
		wantID     int64
	}{
		{
			name: "successful ingredient creation",
			request: PostApiRecipesRecipeIDIngredientsRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					CreateEmptyRecipeIngredient(gomock.Any(), int64(123)).
					Return(database.RecipeIngredient{
						ID:       789,
						RecipeID: 123,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			wantID:     789,
		},
		{
			name: "missing user id in context",
			request: PostApiRecipesRecipeIDIngredientsRequestObject{
				RecipeID: 123,
			},
			injectUser: false,
			setup:      func() {},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			wantID:     0,
		},
		{
			name: "database error on ownership check",
			request: PostApiRecipesRecipeIDIngredientsRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(false, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			wantID:     0,
		},
		{
			name: "user does not own recipe",
			request: PostApiRecipesRecipeIDIngredientsRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(false, nil)
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			wantID:     0,
		},
		{
			name: "database error on ingredient creation",
			request: PostApiRecipesRecipeIDIngredientsRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					CreateEmptyRecipeIngredient(gomock.Any(), int64(123)).
					Return(database.RecipeIngredient{}, errors.New("failed to create ingredient"))
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

			resp, err := server.PostApiRecipesRecipeIDIngredients(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiRecipesRecipeIDIngredients() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PostApiRecipesRecipeIDIngredients200JSONResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if v.Id != tt.wantID {
					t.Errorf("expected ingredient ID %d, got %d", tt.wantID, v.Id)
				}
			case PostApiRecipesRecipeIDIngredients500JSONResponse:
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

func TestPatchApiRecipesRecipeIDIngredientsIngredientID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmoc.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject
		userID     int64
		injectUser bool
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject)
	}{
		{
			name: "successful update with all fields",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name:     stringPtr("Salt"),
					Quantity: float32Ptr(2.5),
					Unit:     stringPtr("tablespoons"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), database.UpdateRecipeIngredientParams{
						ID: 456,
						Name: pgtype.Text{
							String: "Salt",
							Valid:  true,
						},
						Quantity: pgtype.Float4{
							Float32: 2.5,
							Valid:   true,
						},
						Unit: pgtype.Text{
							String: "tablespoons",
							Valid:  true,
						},
					}).
					Return(database.RecipeIngredient{
						ID:       456,
						RecipeID: 123,
						Name: pgtype.Text{
							String: "Salt",
							Valid:  true,
						},
						Quantity: pgtype.Float4{
							Float32: 2.5,
							Valid:   true,
						},
						Unit: pgtype.Text{
							String: "tablespoons",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected ingredient ID 456, got %d", v.Id)
				}
				if v.Name != "Salt" {
					t.Errorf("expected name 'Salt', got %s", v.Name)
				}
				if v.Quantity == nil || *v.Quantity != 2.5 {
					t.Errorf("expected quantity 2.5, got %v", v.Quantity)
				}
				if v.Unit == nil || *v.Unit != "tablespoons" {
					t.Errorf("expected unit 'tablespoons', got %v", v.Unit)
				}
			},
		},
		{
			name: "successful update with only name",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Pepper"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), database.UpdateRecipeIngredientParams{
						ID: 456,
						Name: pgtype.Text{
							String: "Pepper",
							Valid:  true,
						},
					}).
					Return(database.RecipeIngredient{
						ID:       456,
						RecipeID: 123,
						Name: pgtype.Text{
							String: "Pepper",
							Valid:  true,
						},
						Quantity: pgtype.Float4{
							Float32: 1.0,
							Valid:   true,
						},
						Unit: pgtype.Text{
							String: "teaspoon",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse, got %T", resp)
					return
				}
				if v.Name != "Pepper" {
					t.Errorf("expected name 'Pepper', got %s", v.Name)
				}
			},
		},
		{
			name: "successful update with only quantity",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Quantity: float32Ptr(3.0),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), database.UpdateRecipeIngredientParams{
						ID: 456,
						Quantity: pgtype.Float4{
							Float32: 3.0,
							Valid:   true,
						},
					}).
					Return(database.RecipeIngredient{
						ID:       456,
						RecipeID: 123,
						Name: pgtype.Text{
							String: "Salt",
							Valid:  true,
						},
						Quantity: pgtype.Float4{
							Float32: 3.0,
							Valid:   true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse, got %T", resp)
					return
				}
				if v.Quantity == nil || *v.Quantity != 3.0 {
					t.Errorf("expected quantity 3.0, got %v", v.Quantity)
				}
			},
		},
		{
			name: "successful update with only unit",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Unit: stringPtr("cups"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), database.UpdateRecipeIngredientParams{
						ID: 456,
						Unit: pgtype.Text{
							String: "cups",
							Valid:  true,
						},
					}).
					Return(database.RecipeIngredient{
						ID:       456,
						RecipeID: 123,
						Name: pgtype.Text{
							String: "Flour",
							Valid:  true,
						},
						Unit: pgtype.Text{
							String: "cups",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse, got %T", resp)
					return
				}
				if v.Unit == nil || *v.Unit != "cups" {
					t.Errorf("expected unit 'cups', got %v", v.Unit)
				}
			},
		},
		{
			name: "successful update with empty body (no-op)",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body:         &UpdateIngredientBody{},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), database.UpdateRecipeIngredientParams{
						ID: 456,
					}).
					Return(database.RecipeIngredient{
						ID:       456,
						RecipeID: 123,
						Name: pgtype.Text{
							String: "Sugar",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected ingredient ID 456, got %d", v.Id)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Salt"),
				},
			},
			injectUser: false,
			setup:      func() {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
		},
		{
			name: "database error on ownership check",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Salt"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(false, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "user does not own recipe",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Salt"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
		},
		{
			name: "ingredient does not exist in recipe",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 999,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Salt"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), gomock.Any()).
					Return(database.RecipeIngredient{}, pgx.ErrNoRows)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
		},
		{
			name: "database error on update",
			request: PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
				Body: &UpdateIngredientBody{
					Name: stringPtr("Salt"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func() {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), gomock.Any()).
					Return(database.RecipeIngredient{}, errors.New("failed to update ingredient"))
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
			if tt.injectUser {
				ctx = token.UserIDWithCtx(ctx, tt.userID)
			}
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			resp, err := server.PatchApiRecipesRecipeIDIngredientsIngredientID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PatchApiRecipesRecipeIDIngredientsIngredientID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			case PatchApiRecipesRecipeIDIngredientsIngredientID400JSONResponse:
				if tt.wantStatus != 400 {
					t.Errorf("expected status %d, got 400", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PatchApiRecipesRecipeIDIngredientsIngredientID404JSONResponse:
				if tt.wantStatus != 404 {
					t.Errorf("expected status %d, got 404", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PatchApiRecipesRecipeIDIngredientsIngredientID500JSONResponse:
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

// Helper function for creating float32 pointers.
func float32Ptr(f float32) *float32 {
	return &f
}
