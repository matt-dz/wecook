// Package recipes contains handlers for the recipes endpoint.
package recipes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
)

const (
	defaultRecipeTitle = "Untitled Recipe"
)

// CreateRecipe godoc
//
// @Summary Create a recipe.
// @Tags Recipe
// @Succes 200 {object} CreateRecipeResponse
// @Router /api/recipe [POST]
func CreateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	accessToken, err := token.AccessTokenFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract access token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	userIDStr, err := accessToken.Claims.GetSubject()
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get subject from access token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to convert user id to int", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Create recipe
	env.Logger.DebugContext(ctx, "creating recipe")
	recipeID, err := env.Database.CreateRecipe(ctx, database.CreateRecipeParams{
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
		Title: defaultRecipeTitle,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create recipe", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.ErrorContext(ctx, "Writing response")
	resp, err := json.Marshal(CreateRecipeResponse{RecipeID: recipeID})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to marshal response", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		env.Logger.ErrorContext(ctx, "failed to write response", slog.Any("error", err))
		return
	}
}
