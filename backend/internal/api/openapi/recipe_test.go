package client

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
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
	"github.com/matt-dz/wecook/internal/fileserver"
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
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject)
	}{
		{
			name: "missing user id in context",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own recipe",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(GetApiRecipesRecipeID404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "successful recipe retrieval with all fields",
			request: GetApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{
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
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(999)).
					Return(database.GetRecipeAndOwnerRow{}, pgx.ErrNoRows)
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
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{}, errors.New("database connection failed"))
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
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{
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
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{
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
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeAndOwner(gomock.Any(), int64(123)).
					Return(database.GetRecipeAndOwnerRow{
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

func TestGetApiRecipes(t *testing.T) {
	server := NewServer()

	now := time.Now()
	cookTimeUnit := database.TimeUnitMinutes
	prepTimeUnit := database.TimeUnitHours

	tests := []struct {
		name       string
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp GetApiRecipesResponseObject)
	}{
		{
			name:       "missing user id in context",
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesResponseObject) {
				v, ok := resp.(GetApiRecipes400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name:       "database error on getting recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetRecipesByOwner(gomock.Any(), int64(456)).
					Return(nil, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesResponseObject) {
				v, ok := resp.(GetApiRecipes500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name:       "successful retrieval with no recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetRecipesByOwner(gomock.Any(), int64(456)).
					Return([]database.GetRecipesByOwnerRow{}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesResponseObject) {
				v, ok := resp.(GetApiRecipes200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipes200JSONResponse, got %T", resp)
					return
				}
				if len(v.Recipes) != 0 {
					t.Errorf("expected 0 recipes, got %d", len(v.Recipes))
				}
			},
		},
		{
			name:       "successful retrieval with multiple recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetRecipesByOwner(gomock.Any(), int64(456)).
					Return([]database.GetRecipesByOwnerRow{
						{
							UserID:         pgtype.Int8{Int64: 456, Valid: true},
							ImageUrl:       pgtype.Text{String: "recipe1.jpg", Valid: true},
							Title:          "Recipe 1",
							Description:    pgtype.Text{String: "First recipe", Valid: true},
							CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							Published:      true,
							CookTimeAmount: pgtype.Int4{Int32: 30, Valid: true},
							CookTimeUnit:   database.NullTimeUnit{TimeUnit: cookTimeUnit, Valid: true},
							PrepTimeAmount: pgtype.Int4{Int32: 15, Valid: true},
							PrepTimeUnit:   database.NullTimeUnit{TimeUnit: prepTimeUnit, Valid: true},
							RecipeID:       1,
							Servings:       pgtype.Float4{Float32: 4.0, Valid: true},
							FirstName:      "John",
							LastName:       "Doe",
						},
						{
							UserID:         pgtype.Int8{Int64: 456, Valid: true},
							ImageUrl:       pgtype.Text{String: "", Valid: false},
							Title:          "Recipe 2",
							Description:    pgtype.Text{String: "", Valid: false},
							CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							Published:      false,
							CookTimeAmount: pgtype.Int4{Int32: 0, Valid: false},
							CookTimeUnit:   database.NullTimeUnit{TimeUnit: "", Valid: false},
							PrepTimeAmount: pgtype.Int4{Int32: 0, Valid: false},
							PrepTimeUnit:   database.NullTimeUnit{TimeUnit: "", Valid: false},
							RecipeID:       2,
							Servings:       pgtype.Float4{Float32: 0, Valid: false},
							FirstName:      "John",
							LastName:       "Doe",
						},
					}, nil)

				mockFS.EXPECT().
					FileURL("recipe1.jpg").
					Return("http://test-host/recipe1.jpg")
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesResponseObject) {
				v, ok := resp.(GetApiRecipes200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipes200JSONResponse, got %T", resp)
					return
				}
				if len(v.Recipes) != 2 {
					t.Errorf("expected 2 recipes, got %d", len(v.Recipes))
					return
				}

				// Validate first recipe (with all fields)
				recipe1 := v.Recipes[0]
				if recipe1.Recipe.Id != 1 {
					t.Errorf("expected recipe ID 1, got %d", recipe1.Recipe.Id)
				}
				if recipe1.Recipe.Title != "Recipe 1" {
					t.Errorf("expected title 'Recipe 1', got %s", recipe1.Recipe.Title)
				}
				if recipe1.Recipe.Description == nil || *recipe1.Recipe.Description != "First recipe" {
					t.Errorf("expected description 'First recipe', got %v", recipe1.Recipe.Description)
				}
				if recipe1.Recipe.ImageUrl == nil || *recipe1.Recipe.ImageUrl != "http://test-host/recipe1.jpg" {
					t.Errorf("expected image URL 'http://test-host/recipe1.jpg', got %v", recipe1.Recipe.ImageUrl)
				}
				if !recipe1.Recipe.Published {
					t.Errorf("expected recipe to be published")
				}
				if recipe1.Recipe.CookTimeAmount == nil || *recipe1.Recipe.CookTimeAmount != 30 {
					t.Errorf("expected cook time amount 30, got %v", recipe1.Recipe.CookTimeAmount)
				}
				if recipe1.Recipe.PrepTimeAmount == nil || *recipe1.Recipe.PrepTimeAmount != 15 {
					t.Errorf("expected prep time amount 15, got %v", recipe1.Recipe.PrepTimeAmount)
				}
				if recipe1.Recipe.Servings == nil || *recipe1.Recipe.Servings != 4.0 {
					t.Errorf("expected servings 4.0, got %v", recipe1.Recipe.Servings)
				}
				if recipe1.Owner.FirstName != "John" {
					t.Errorf("expected owner first name 'John', got %s", recipe1.Owner.FirstName)
				}
				if recipe1.Owner.LastName != "Doe" {
					t.Errorf("expected owner last name 'Doe', got %s", recipe1.Owner.LastName)
				}

				// Validate second recipe (minimal fields)
				recipe2 := v.Recipes[1]
				if recipe2.Recipe.Id != 2 {
					t.Errorf("expected recipe ID 2, got %d", recipe2.Recipe.Id)
				}
				if recipe2.Recipe.Title != "Recipe 2" {
					t.Errorf("expected title 'Recipe 2', got %s", recipe2.Recipe.Title)
				}
				if recipe2.Recipe.Description != nil {
					t.Errorf("expected nil description, got %v", recipe2.Recipe.Description)
				}
				if recipe2.Recipe.ImageUrl != nil {
					t.Errorf("expected nil image URL, got %v", recipe2.Recipe.ImageUrl)
				}
				if recipe2.Recipe.Published {
					t.Errorf("expected recipe to not be published")
				}
				if recipe2.Recipe.CookTimeAmount != nil {
					t.Errorf("expected nil cook time amount, got %v", recipe2.Recipe.CookTimeAmount)
				}
				if recipe2.Recipe.PrepTimeAmount != nil {
					t.Errorf("expected nil prep time amount, got %v", recipe2.Recipe.PrepTimeAmount)
				}
				if recipe2.Recipe.Servings != nil {
					t.Errorf("expected nil servings, got %v", recipe2.Recipe.Servings)
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

			resp, err := server.GetApiRecipes(ctx, GetApiRecipesRequestObject{})
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiRecipes() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestGetApiRecipesPublic(t *testing.T) {
	server := NewServer()

	now := time.Now()
	cookTimeUnit := database.TimeUnitMinutes
	prepTimeUnit := database.TimeUnitHours

	tests := []struct {
		name       string
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp GetApiRecipesPublicResponseObject)
	}{
		{
			name:       "database error on getting recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublicRecipes(gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesPublicResponseObject) {
				v, ok := resp.(GetApiRecipesPublic500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name:       "successful retrieval with no recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublicRecipes(gomock.Any()).
					Return([]database.GetPublicRecipesRow{}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesPublicResponseObject) {
				v, ok := resp.(GetApiRecipesPublic200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesPublic200JSONResponse, got %T", resp)
					return
				}
				if len(v.Recipes) != 0 {
					t.Errorf("expected 0 recipes, got %d", len(v.Recipes))
				}
			},
		},
		{
			name:       "successful retrieval with multiple recipes",
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					GetPublicRecipes(gomock.Any()).
					Return([]database.GetPublicRecipesRow{
						{
							UserID:         pgtype.Int8{Int64: 456, Valid: true},
							ImageUrl:       pgtype.Text{String: "recipe1.jpg", Valid: true},
							Title:          "Recipe 1",
							Description:    pgtype.Text{String: "First recipe", Valid: true},
							CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							Published:      true,
							CookTimeAmount: pgtype.Int4{Int32: 30, Valid: true},
							CookTimeUnit:   database.NullTimeUnit{TimeUnit: cookTimeUnit, Valid: true},
							PrepTimeAmount: pgtype.Int4{Int32: 15, Valid: true},
							PrepTimeUnit:   database.NullTimeUnit{TimeUnit: prepTimeUnit, Valid: true},
							RecipeID:       1,
							Servings:       pgtype.Float4{Float32: 4.0, Valid: true},
							FirstName:      "John",
							LastName:       "Doe",
						},
						{
							UserID:         pgtype.Int8{Int64: 456, Valid: true},
							ImageUrl:       pgtype.Text{String: "", Valid: false},
							Title:          "Recipe 2",
							Description:    pgtype.Text{String: "", Valid: false},
							CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
							Published:      false,
							CookTimeAmount: pgtype.Int4{Int32: 0, Valid: false},
							CookTimeUnit:   database.NullTimeUnit{TimeUnit: "", Valid: false},
							PrepTimeAmount: pgtype.Int4{Int32: 0, Valid: false},
							PrepTimeUnit:   database.NullTimeUnit{TimeUnit: "", Valid: false},
							RecipeID:       2,
							Servings:       pgtype.Float4{Float32: 0, Valid: false},
							FirstName:      "John",
							LastName:       "Doe",
						},
					}, nil)

				mockFS.EXPECT().
					FileURL("recipe1.jpg").
					Return("http://test-host/recipe1.jpg")
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp GetApiRecipesPublicResponseObject) {
				v, ok := resp.(GetApiRecipesPublic200JSONResponse)
				if !ok {
					t.Errorf("expected GetApiRecipesPublic200JSONResponse, got %T", resp)
					return
				}
				if len(v.Recipes) != 2 {
					t.Errorf("expected 2 recipes, got %d", len(v.Recipes))
					return
				}

				// Validate first recipe (with all fields)
				recipe1 := v.Recipes[0]
				if recipe1.Recipe.Id != 1 {
					t.Errorf("expected recipe ID 1, got %d", recipe1.Recipe.Id)
				}
				if recipe1.Recipe.Title != "Recipe 1" {
					t.Errorf("expected title 'Recipe 1', got %s", recipe1.Recipe.Title)
				}
				if recipe1.Recipe.Description == nil || *recipe1.Recipe.Description != "First recipe" {
					t.Errorf("expected description 'First recipe', got %v", recipe1.Recipe.Description)
				}
				if recipe1.Recipe.ImageUrl == nil || *recipe1.Recipe.ImageUrl != "http://test-host/recipe1.jpg" {
					t.Errorf("expected image URL 'http://test-host/recipe1.jpg', got %v", recipe1.Recipe.ImageUrl)
				}
				if !recipe1.Recipe.Published {
					t.Errorf("expected recipe to be published")
				}
				if recipe1.Recipe.CookTimeAmount == nil || *recipe1.Recipe.CookTimeAmount != 30 {
					t.Errorf("expected cook time amount 30, got %v", recipe1.Recipe.CookTimeAmount)
				}
				if recipe1.Recipe.PrepTimeAmount == nil || *recipe1.Recipe.PrepTimeAmount != 15 {
					t.Errorf("expected prep time amount 15, got %v", recipe1.Recipe.PrepTimeAmount)
				}
				if recipe1.Recipe.Servings == nil || *recipe1.Recipe.Servings != 4.0 {
					t.Errorf("expected servings 4.0, got %v", recipe1.Recipe.Servings)
				}
				if recipe1.Owner.FirstName != "John" {
					t.Errorf("expected owner first name 'John', got %s", recipe1.Owner.FirstName)
				}
				if recipe1.Owner.LastName != "Doe" {
					t.Errorf("expected owner last name 'Doe', got %s", recipe1.Owner.LastName)
				}

				// Validate second recipe (minimal fields)
				recipe2 := v.Recipes[1]
				if recipe2.Recipe.Id != 2 {
					t.Errorf("expected recipe ID 2, got %d", recipe2.Recipe.Id)
				}
				if recipe2.Recipe.Title != "Recipe 2" {
					t.Errorf("expected title 'Recipe 2', got %s", recipe2.Recipe.Title)
				}
				if recipe2.Recipe.Description != nil {
					t.Errorf("expected nil description, got %v", recipe2.Recipe.Description)
				}
				if recipe2.Recipe.ImageUrl != nil {
					t.Errorf("expected nil image URL, got %v", recipe2.Recipe.ImageUrl)
				}
				if recipe2.Recipe.Published {
					t.Errorf("expected recipe to not be published")
				}
				if recipe2.Recipe.CookTimeAmount != nil {
					t.Errorf("expected nil cook time amount, got %v", recipe2.Recipe.CookTimeAmount)
				}
				if recipe2.Recipe.PrepTimeAmount != nil {
					t.Errorf("expected nil prep time amount, got %v", recipe2.Recipe.PrepTimeAmount)
				}
				if recipe2.Recipe.Servings != nil {
					t.Errorf("expected nil servings, got %v", recipe2.Recipe.Servings)
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

			resp, err := server.GetApiRecipesPublic(ctx, GetApiRecipesPublicRequestObject{})
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiRecipes() error = %v, wantError %v", err, tt.wantError)
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 999,
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
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
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

func TestPostApiRecipesRecipeIDIngredientsIngredientIDImage(t *testing.T) {
	// Create a simple PNG image for testing (1x1 pixel)
	validPNGImage := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Create a simple JPEG image header for testing
	validJPEGImage := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
		0xFF, 0xD9, // EOI
	}

	invalidImage := []byte("not an image")

	tests := []struct {
		name       string
		request    PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject
		userID     int64
		injectUser bool
		imageData  []byte
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject)
	}{
		{
			name: "successful upload without existing image",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteIngredientImage(int64(123), int64(456), ".png", validPNGImage).
					Return("files/ingredients/123/456.png", len(validPNGImage), nil)

				mockFS.EXPECT().
					FileURL("files/ingredients/123/456.png").
					Return("http://test-host/files/ingredients/123/456.png")

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), gomock.Any()).
					Return(database.RecipeIngredient{
						ID: 456,
						ImageUrl: pgtype.Text{
							String: "files/ingredients/123/456.png",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.ImageUrl == nil {
					t.Errorf("expected non-nil image_url")
				}
				if *v.ImageUrl != "http://test-host/files/ingredients/123/456.png" {
					t.Errorf("expected image_url 'http://test-host/files/ingredients/123/456.png', got %s", *v.ImageUrl)
				}
			},
		},
		{
			name: "successful upload replacing existing image",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validJPEGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456-old.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456-old.png").
					Return(nil)

				mockFS.EXPECT().
					WriteIngredientImage(int64(123), int64(456), ".jpg", validJPEGImage).
					Return("files/ingredients/123/456.jpg", len(validJPEGImage), nil)

				mockFS.EXPECT().
					FileURL("files/ingredients/123/456.jpg").
					Return("http://test-host/files/ingredients/123/456.jpg")

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), gomock.Any()).
					Return(database.RecipeIngredient{
						ID: 456,
						ImageUrl: pgtype.Text{
							String: "files/ingredients/123/456.jpg",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.ImageUrl == nil {
					t.Errorf("expected non-nil image_url")
				}
				if *v.ImageUrl != "http://test-host/files/ingredients/123/456.jpg" {
					t.Errorf("expected image_url 'http://test-host/files/ingredients/123/456.jpg', got %s", *v.ImageUrl)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     0,
			injectUser: false,
			imageData:  validPNGImage,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own recipe",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "invalid image format",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  invalidImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)
			},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting old image URL",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "error deleting old image",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456-old.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456-old.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "error writing new image",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteIngredientImage(int64(123), int64(456), ".png", validPNGImage).
					Return("", 0, errors.New("file write error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "database error updating ingredient",
			request: PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteIngredientImage(int64(123), int64(456), ".png", validPNGImage).
					Return("files/ingredients/123/456.png", len(validPNGImage), nil)

				mockDB.EXPECT().
					UpdateRecipeIngredient(gomock.Any(), gomock.Any()).
					Return(database.RecipeIngredient{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			// Create multipart form with image
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("image", "test.png")
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			_, err = part.Write(tt.imageData)
			if err != nil {
				t.Fatalf("failed to write image data: %v", err)
			}
			err = writer.Close()
			if err != nil {
				t.Logf("failed to close writer: %s", err.Error())
			}

			// Create multipart reader
			reader := multipart.NewReader(body, writer.Boundary())

			tt.request.Body = reader

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

			server := NewServer()
			resp, err := server.PostApiRecipesRecipeIDIngredientsIngredientIDImage(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiRecipesRecipeIDIngredientsIngredientIDImage() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestDeleteApiRecipesRecipeIDIngredientsIngredientIDImage(t *testing.T) {
	tests := []struct {
		name       string
		request    DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject)
	}{
		{
			name: "successful deletion",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456.png").
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "missing user id in context",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own ingredient",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting image URL",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "no image exists to delete",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.ImageNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.ImageNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.ImageNotFound.String(), v.Code)
				}
				if v.Message != "image not found" {
					t.Errorf("expected message 'image not found', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on delete from database",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "file system error deleting image",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			server := NewServer()
			resp, err := server.DeleteApiRecipesRecipeIDIngredientsIngredientIDImage(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteApiRecipesRecipeIDIngredientsIngredientIDImage() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestPostApiRecipesRecipeIDSteps(t *testing.T) {
	tests := []struct {
		name       string
		request    PostApiRecipesRecipeIDStepsRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject)
	}{
		{
			name: "successful step creation",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					CreateRecipeStep(gomock.Any(), database.CreateRecipeStepParams{
						RecipeID: 123,
					}).
					Return(database.CreateRecipeStepRow{
						ID:         456,
						StepNumber: 1,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.StepNumber != 1 {
					t.Errorf("expected step_number 1, got %d", v.StepNumber)
				}
			},
		},
		{
			name: "successful step creation with step number 2",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					CreateRecipeStep(gomock.Any(), database.CreateRecipeStepParams{
						RecipeID: 123,
					}).
					Return(database.CreateRecipeStepRow{
						ID:         457,
						StepNumber: 2,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 457 {
					t.Errorf("expected id 457, got %d", v.Id)
				}
				if v.StepNumber != 2 {
					t.Errorf("expected step_number 2, got %d", v.StepNumber)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
				if v.Message != "missing user id" {
					t.Errorf("expected message 'missing user id', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own recipe",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
				if v.Message != "recipe does not exist or user does not own recipe" {
					t.Errorf("expected message 'recipe does not exist or user does not own recipe', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on step creation",
			request: PostApiRecipesRecipeIDStepsRequestObject{
				RecipeID: 123,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					CreateRecipeStep(gomock.Any(), database.CreateRecipeStepParams{
						RecipeID: 123,
					}).
					Return(database.CreateRecipeStepRow{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDSteps500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
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

			server := NewServer()
			resp, err := server.PostApiRecipesRecipeIDSteps(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiRecipesRecipeIDSteps() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

// Helper function for creating int32 pointers.
func int32Ptr(i int32) *int32 {
	return &i
}

func TestPatchApiRecipesRecipeIDStepsStepID(t *testing.T) {
	tests := []struct {
		name       string
		request    PatchApiRecipesRecipeIDStepsStepIDRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject)
	}{
		{
			name: "successful update with instruction only",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Mix all ingredients together"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), database.UpdateRecipeStepParams{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Mix all ingredients together",
							Valid:  true,
						},
					}).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Mix all ingredients together",
							Valid:  true,
						},
						StepNumber: 1,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.Instruction == nil || *v.Instruction != "Mix all ingredients together" {
					t.Errorf("expected instruction 'Mix all ingredients together', got %v", v.Instruction)
				}
				if v.StepNumber != 1 {
					t.Errorf("expected step_number 1, got %d", v.StepNumber)
				}
			},
		},
		{
			name: "successful update with step_number only",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					StepNumber: int32Ptr(3),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), database.UpdateRecipeStepParams{
						ID: 456,
						StepNumber: pgtype.Int4{
							Int32: 3,
							Valid: true,
						},
					}).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Existing instruction",
							Valid:  true,
						},
						StepNumber: 3,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.StepNumber != 3 {
					t.Errorf("expected step_number 3, got %d", v.StepNumber)
				}
			},
		},
		{
			name: "successful update with both instruction and step_number",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Bake for 30 minutes"),
					StepNumber:  int32Ptr(2),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockFS.EXPECT().
					FileURL("files/steps/123/456.png").
					Return("http://test-host/files/steps/123/456.png")

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), database.UpdateRecipeStepParams{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Bake for 30 minutes",
							Valid:  true,
						},
						StepNumber: pgtype.Int4{
							Int32: 2,
							Valid: true,
						},
					}).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Bake for 30 minutes",
							Valid:  true,
						},
						StepNumber: 2,
						ImageUrl: pgtype.Text{
							String: "files/steps/123/456.png",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.Instruction == nil || *v.Instruction != "Bake for 30 minutes" {
					t.Errorf("expected instruction 'Bake for 30 minutes', got %v", v.Instruction)
				}
				if v.StepNumber != 2 {
					t.Errorf("expected step_number 2, got %d", v.StepNumber)
				}
				if v.ImageUrl == nil || *v.ImageUrl != "http://test-host/files/steps/123/456.png" {
					t.Errorf("expected image_url 'http://test-host/files/steps/123/456.png', got %v", v.ImageUrl)
				}
			},
		},
		{
			name: "successful update with empty body (no-op)",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body:     &UpdateStepRequest{},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), database.UpdateRecipeStepParams{
						ID: 456,
					}).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						Instruction: pgtype.Text{
							String: "Unchanged instruction",
							Valid:  true,
						},
						StepNumber: 1,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Some instruction"),
				},
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
				if v.Message != "missing user id" {
					t.Errorf("expected message 'missing user id', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Some instruction"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own step",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Some instruction"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
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
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
				if v.Message != "recipe/step does not exist or user does not own recipe" {
					t.Errorf("expected message 'recipe/step does not exist or user does not own recipe', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on update",
			request: PatchApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
				Body: &UpdateStepRequest{
					Instruction: stringPtr("Some instruction"),
				},
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeStepRow{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
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

			server := NewServer()
			resp, err := server.PatchApiRecipesRecipeIDStepsStepID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PatchApiRecipesRecipeIDStepsStepID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestPostApiRecipesRecipeIDStepsStepIDImage(t *testing.T) {
	// Create a simple PNG image for testing (1x1 pixel)
	validPNGImage := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xDD, 0x8D,
		0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Create a simple JPEG image header for testing
	validJPEGImage := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08,
		0xFF, 0xD9, // EOI
	}

	invalidImage := []byte("not an image")

	tests := []struct {
		name       string
		request    PostApiRecipesRecipeIDStepsStepIDImageRequestObject
		userID     int64
		injectUser bool
		imageData  []byte
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject)
	}{
		{
			name: "successful upload without existing image",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteStepImage(int64(123), int64(456), ".png", validPNGImage).
					Return("files/steps/123/456.png", len(validPNGImage), nil)

				mockFS.EXPECT().
					FileURL("files/steps/123/456.png").
					Return("http://test-host/files/steps/123/456.png")

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						ImageUrl: pgtype.Text{
							String: "files/steps/123/456.png",
							Valid:  true,
						},
						StepNumber: 1,
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.Id != 456 {
					t.Errorf("expected id 456, got %d", v.Id)
				}
				if v.ImageUrl == nil || *v.ImageUrl != "http://test-host/files/steps/123/456.png" {
					t.Errorf("expected image_url 'http://test-host/files/steps/123/456.png', got %v", v.ImageUrl)
				}
			},
		},
		{
			name: "successful upload replacing existing image",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validJPEGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456-old.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456-old.png").
					Return(nil)

				mockFS.EXPECT().
					WriteStepImage(int64(123), int64(456), ".jpg", validJPEGImage).
					Return("files/steps/123/456.jpg", len(validJPEGImage), nil)

				mockFS.EXPECT().
					FileURL("files/steps/123/456.jpg").
					Return("http://test-host/files/steps/123/456.jpg")

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeStepRow{
						ID: 456,
						ImageUrl: pgtype.Text{
							String: "files/steps/123/456.jpg",
							Valid:  true,
						},
						StepNumber: 2,
						Instruction: pgtype.Text{
							String: "Mix ingredients",
							Valid:  true,
						},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage200JSONResponse)
				if !ok {
					t.Errorf("expected 200 response, got %T", resp)
					return
				}
				if v.ImageUrl == nil || *v.ImageUrl != "http://test-host/files/steps/123/456.jpg" {
					t.Errorf("expected image_url 'http://test-host/files/steps/123/456.jpg', got %v", v.ImageUrl)
				}
				if v.Instruction == nil || *v.Instruction != "Mix ingredients" {
					t.Errorf("expected instruction 'Mix ingredients', got %v", v.Instruction)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     0,
			injectUser: false,
			imageData:  validPNGImage,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own step",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "invalid image format",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  invalidImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)
			},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting old image URL",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "error deleting old image",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456-old.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456-old.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "error writing new image",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteStepImage(int64(123), int64(456), ".png", validPNGImage).
					Return("", 0, errors.New("file write error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "database error updating step",
			request: PostApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			imageData:  validPNGImage,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockFS.EXPECT().
					WriteStepImage(int64(123), int64(456), ".png", validPNGImage).
					Return("files/steps/123/456.png", len(validPNGImage), nil)

				mockDB.EXPECT().
					UpdateRecipeStep(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeStepRow{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PostApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			// Create multipart form with image
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("image", "test.png")
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			_, err = part.Write(tt.imageData)
			if err != nil {
				t.Fatalf("failed to write image data: %v", err)
			}
			err = writer.Close()
			if err != nil {
				t.Logf("failed to close writer: %s", err.Error())
			}

			// Create multipart reader
			reader := multipart.NewReader(body, writer.Boundary())

			tt.request.Body = reader

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

			server := NewServer()
			resp, err := server.PostApiRecipesRecipeIDStepsStepIDImage(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiRecipesRecipeIDStepsStepIDImage() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestDeleteApiRecipesRecipeIDStepsStepIDImage(t *testing.T) {
	tests := []struct {
		name       string
		request    DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject)
	}{
		{
			name: "successful deletion",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456.png").
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "missing user id in context",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own step",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting image URL",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "no image exists to delete",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.ImageNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.ImageNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.ImageNotFound.String(), v.Code)
				}
				if v.Message != "image not found" {
					t.Errorf("expected message 'image not found', got %s", v.Message)
				}
			},
		},
		{
			name: "database error on delete from database",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "file system error deleting image",
			request: DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockDB.EXPECT().
					DeleteRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			server := NewServer()
			resp, err := server.DeleteApiRecipesRecipeIDStepsStepIDImage(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteApiRecipesRecipeIDStepsStepIDImage() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestDeleteApiRecipesRecipeIDIngredientsIngredientID(t *testing.T) {
	tests := []struct {
		name       string
		request    DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject)
	}{
		{
			name: "successful deletion without image",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockDB.EXPECT().
					DeleteRecipeIngredient(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "successful deletion with image",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456.png").
					Return(nil)

				mockDB.EXPECT().
					DeleteRecipeIngredient(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "successful deletion with missing image file",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), database.CheckIngredientOwnershipParams{
						RecipeID:     123,
						IngredientID: 456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456.png").
					Return(fileserver.ErrNotExist)

				mockDB.EXPECT().
					DeleteRecipeIngredient(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "missing user id in context",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own ingredient",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting image URL",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "file system error deleting image",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/ingredients/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/ingredients/123/456.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "database error deleting ingredient",
			request: DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject{
				RecipeID:     123,
				IngredientID: 456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckIngredientOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeIngredientImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockDB.EXPECT().
					DeleteRecipeIngredient(gomock.Any(), int64(456)).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			server := NewServer()
			resp, err := server.DeleteApiRecipesRecipeIDIngredientsIngredientID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteApiRecipesRecipeIDIngredientsIngredientID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestDeleteApiRecipesRecipeIDStepsStepID(t *testing.T) {
	tests := []struct {
		name       string
		request    DeleteApiRecipesRecipeIDStepsStepIDRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject)
	}{
		{
			name: "successful deletion without image",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockDB.EXPECT().
					DeleteRecipeStep(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "successful deletion with image",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456.png").
					Return(nil)

				mockDB.EXPECT().
					DeleteRecipeStep(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "successful deletion with missing image file",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), database.CheckStepOwnershipParams{
						RecipeID: 123,
						StepID:   456,
						UserID: pgtype.Int8{
							Int64: 789,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456.png").
					Return(fileserver.ErrNotExist)

				mockDB.EXPECT().
					DeleteRecipeStep(gomock.Any(), int64(456)).
					Return(nil)
			},
			wantStatus: 204,
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID204Response)
				if !ok {
					t.Errorf("expected 204 response, got %T", resp)
				}
			},
		},
		{
			name: "missing user id in context",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     0,
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID400JSONResponse)
				if !ok {
					t.Errorf("expected 400 response, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own step",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(false, nil)
			},
			wantStatus: 404,
			wantCode:   apiError.RecipeNotFound.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				v, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID404JSONResponse)
				if !ok {
					t.Errorf("expected 404 response, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error getting image URL",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{}, errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "file system error deleting image",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{
						String: "files/steps/123/456.png",
						Valid:  true,
					}, nil)

				mockFS.EXPECT().
					DeleteURLPath("files/steps/123/456.png").
					Return(errors.New("file system error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
				}
			},
		},
		{
			name: "database error deleting step",
			request: DeleteApiRecipesRecipeIDStepsStepIDRequestObject{
				RecipeID: 123,
				StepID:   456,
			},
			userID:     789,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckStepOwnership(gomock.Any(), gomock.Any()).
					Return(true, nil)

				mockDB.EXPECT().
					GetRecipeStepImageURL(gomock.Any(), int64(456)).
					Return(pgtype.Text{Valid: false}, nil)

				mockDB.EXPECT().
					DeleteRecipeStep(gomock.Any(), int64(456)).
					Return(errors.New("database error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp DeleteApiRecipesRecipeIDStepsStepIDResponseObject) {
				_, ok := resp.(DeleteApiRecipesRecipeIDStepsStepID500JSONResponse)
				if !ok {
					t.Errorf("expected 500 response, got %T", resp)
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

			server := NewServer()
			resp, err := server.DeleteApiRecipesRecipeIDStepsStepID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("DeleteApiRecipesRecipeIDStepsStepID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func TestPatchApiRecipesRecipeID(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		request    PatchApiRecipesRecipeIDRequestObject
		userID     int64
		injectUser bool
		setup      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface)
		wantStatus int
		wantCode   string
		wantError  bool
		validate   func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject)
	}{
		{
			name: "successful update with all fields",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Title:          stringPtr("Updated Recipe"),
					Description:    stringPtr("An updated description"),
					Servings:       float32Ptr(6.0),
					CookTimeAmount: int32Ptr(45),
					CookTimeUnit:   timeUnitPtr(Minutes),
					PrepTimeAmount: int32Ptr(20),
					PrepTimeUnit:   timeUnitPtr(Minutes),
					Published:      boolPtr(true),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
				mockDB.EXPECT().
					CheckRecipeOwnership(gomock.Any(), database.CheckRecipeOwnershipParams{
						ID: 123,
						UserID: pgtype.Int8{
							Int64: 456,
							Valid: true,
						},
					}).
					Return(true, nil)

				mockFS.EXPECT().
					FileURL("recipe.jpg").
					Return("http://test-host/recipe.jpg")

				mockDB.EXPECT().
					UpdateRecipe(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeRow{
						ID:             123,
						Title:          "Updated Recipe",
						Description:    pgtype.Text{String: "An updated description", Valid: true},
						Servings:       pgtype.Float4{Float32: 6.0, Valid: true},
						CookTimeAmount: pgtype.Int4{Int32: 45, Valid: true},
						CookTimeUnit:   database.NullTimeUnit{TimeUnit: database.TimeUnitMinutes, Valid: true},
						PrepTimeAmount: pgtype.Int4{Int32: 20, Valid: true},
						PrepTimeUnit:   database.NullTimeUnit{TimeUnit: database.TimeUnitMinutes, Valid: true},
						Published:      true,
						CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
						ImageUrl:       pgtype.Text{String: "recipe.jpg", Valid: true},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID200JSONResponse, got %T", resp)
					return
				}
				if v.Id != 123 {
					t.Errorf("expected ID 123, got %d", v.Id)
				}
				if v.Title != "Updated Recipe" {
					t.Errorf("expected title 'Updated Recipe', got %s", v.Title)
				}
				if v.Description == nil || *v.Description != "An updated description" {
					t.Errorf("expected description 'An updated description', got %v", v.Description)
				}
				if v.Servings == nil || *v.Servings != 6.0 {
					t.Errorf("expected servings 6.0, got %v", v.Servings)
				}
				if v.CookTimeAmount == nil || *v.CookTimeAmount != 45 {
					t.Errorf("expected cook time amount 45, got %v", v.CookTimeAmount)
				}
				if v.PrepTimeAmount == nil || *v.PrepTimeAmount != 20 {
					t.Errorf("expected prep time amount 20, got %v", v.PrepTimeAmount)
				}
				if !v.Published {
					t.Errorf("expected published to be true")
				}
			},
		},
		{
			name: "successful update with partial fields",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Title:     stringPtr("New Title"),
					Published: boolPtr(false),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					UpdateRecipe(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeRow{
						ID:        123,
						Title:     "New Title",
						Published: false,
						CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID200JSONResponse, got %T", resp)
					return
				}
				if v.Title != "New Title" {
					t.Errorf("expected title 'New Title', got %s", v.Title)
				}
				if v.Published {
					t.Errorf("expected published to be false")
				}
			},
		},
		{
			name: "successful update with only description",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Description: stringPtr("Just updating description"),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					UpdateRecipe(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeRow{
						ID:          123,
						Title:       "Original Title",
						Description: pgtype.Text{String: "Just updating description", Valid: true},
						Published:   false,
						CreatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
						UpdatedAt:   pgtype.Timestamptz{Time: now, Valid: true},
					}, nil)
			},
			wantStatus: 200,
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID200JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID200JSONResponse, got %T", resp)
					return
				}
				if v.Description == nil || *v.Description != "Just updating description" {
					t.Errorf("expected description 'Just updating description', got %v", v.Description)
				}
			},
		},
		{
			name: "missing user id in context",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body:     &PatchApiRecipesRecipeIDJSONRequestBody{},
			},
			injectUser: false,
			setup:      func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID400JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID400JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.BadRequest.String() {
					t.Errorf("expected code %s, got %s", apiError.BadRequest.String(), v.Code)
				}
			},
		},
		{
			name: "database error on ownership check",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Title: stringPtr("New Title"),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID500JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
				}
			},
		},
		{
			name: "user does not own recipe",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Title: stringPtr("New Title"),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID404JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID404JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.RecipeNotFound.String() {
					t.Errorf("expected code %s, got %s", apiError.RecipeNotFound.String(), v.Code)
				}
			},
		},
		{
			name: "database error on update",
			request: PatchApiRecipesRecipeIDRequestObject{
				RecipeID: 123,
				Body: &PatchApiRecipesRecipeIDJSONRequestBody{
					Title: stringPtr("New Title"),
				},
			},
			userID:     456,
			injectUser: true,
			setup: func(mockDB *dbmoc.MockQuerier, mockFS *filestore.MockFileStoreInterface) {
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
					UpdateRecipe(gomock.Any(), gomock.Any()).
					Return(database.UpdateRecipeRow{}, errors.New("database connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
			validate: func(t *testing.T, resp PatchApiRecipesRecipeIDResponseObject) {
				v, ok := resp.(PatchApiRecipesRecipeID500JSONResponse)
				if !ok {
					t.Errorf("expected PatchApiRecipesRecipeID500JSONResponse, got %T", resp)
					return
				}
				if v.Code != apiError.InternalServerError.String() {
					t.Errorf("expected code %s, got %s", apiError.InternalServerError.String(), v.Code)
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
			server := NewServer()

			tt.setup(mockDB, mockFS)

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

			resp, err := server.PatchApiRecipesRecipeID(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PatchApiRecipesRecipeID() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.validate != nil {
				tt.validate(t, resp)
			}
		})
	}
}

func timeUnitPtr(t TimeUnit) *TimeUnit {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}
