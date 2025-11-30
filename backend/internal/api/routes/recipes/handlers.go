// Package recipes contains handlers for the recipes endpoint.
package recipes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/fileserver"
	"github.com/matt-dz/wecook/internal/recipe"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultRecipeTitle = "Untitled Recipe"
	maxUploadSize      = 20 << 20 // ~ 20 MB
)

// CreateRecipe godoc
//
//	@Summary	Create a recipe.
//	@Tags		Recipe
//	@Succes		200 {object} CreateRecipeResponse
//	@Router		/api/recipes [POST]
func CreateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
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

// CreateRecipeIngredient godoc
//
//	@Summary		Create a recipe ingredient
//	@Description	Creates a new ingredient for a recipe owned by the authenticated user.
//	@Description	Expects multipart/form-data with basic fields and an optional image upload.
//	@Tags			Recipe, Ingredient
//	@Accept			multipart/form-data
//	@Produce		json
//
//	@Param			recipe-id	formData	string	true	"Recipe ID the ingredient belongs to"
//	@Param			name		formData	string	true	"Ingredient name"
//	@Param			quantity	formData	number	true	"Ingredient quantity"
//	@Param			unit		formData	string	false	"Ingredient unit (e.g. g, ml, tbsp)"
//	@Param			image		formData	file	false	"Ingredient image (JPEG/PNG)"
//
//	@Success		201			"Ingredient created"
//	@Failure		400			{object}	apiError.Error	"Bad request / validation error / unsupported file type"
//	@Failure		401			{object}	apiError.Error	"Unauthorized"
//	@Failure		403			{object}	apiError.Error	"User does not own recipe"
//	@Failure		404			{object}	apiError.Error	"Recipe not found"
//	@Failure		500			{object}	apiError.Error	"Internal server error"
//
//	@Security		AccessTokenCookie
//	@Router			/api/recipes/ingredients [post]
func CreateRecipeIngredient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read response
	env.Logger.DebugContext(ctx, "Reading response")
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		env.Logger.ErrorContext(ctx, "failed to parse multipart form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "request too large", requestID)
		return
	}
	request := CreateIngredientRequest{
		RecipeID: recipeID(strings.TrimSpace(r.Form.Get("recipe-id"))),
		Name:     strings.TrimSpace(r.Form.Get("name")),
		Quantity: quantity(strings.TrimSpace(r.Form.Get("quantity"))),
		Unit:     strings.TrimSpace(r.Form.Get("unit")),
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	uploadedImage, err := recipe.ReadIngredientImage(r)
	if errors.Is(err, recipe.ErrNoImageUploaded) {
		env.Logger.DebugContext(ctx, "no image uploaded")
	} else if errors.Is(err, recipe.ErrUnsupportedMimeType) {
		env.Logger.ErrorContext(ctx, "unsupported file type", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid file type", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read ingredient image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Get Recipe Owner
	recipeID, _ := strconv.ParseInt(string(request.RecipeID), 10, 64)
	env.Logger.ErrorContext(ctx, "Getting recipe owner")
	owner, err := env.Database.GetRecipeOwner(ctx, recipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "recipe does not exists")
		_ = apiError.EncodeError(w, apiError.RecipeNotFound, "recipe not found", requestID)
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe owner")
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	if owner.Int64 != userID {
		env.Logger.ErrorContext(ctx, "user does not own recipe", slog.Any("owner-id", owner.Int64))
		_ = apiError.EncodeError(w, apiError.InsufficientPermissions, "cannot add ingredient", requestID)
		return
	}

	// Adding recipe ingredient
	env.Logger.DebugContext(ctx, "creating recipe ingredient")
	quantity, _ := strconv.ParseFloat(string(request.Quantity), 32)
	ingredientID, err := env.Database.CreateRecipeIngredient(ctx, database.CreateRecipeIngredientParams{
		RecipeID: recipeID,
		Quantity: float32(quantity),
		Unit: pgtype.Text{
			String: request.Unit,
			Valid:  request.Unit == "",
		},
		Name:     request.Name,
		ImageUrl: pgtype.Text{Valid: false},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create recipe ingredient", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Upload recipe image
	env.Logger.DebugContext(ctx, "uploading recipe image")
	path := filepath.Join(fileserver.IngredientsDir,
		strconv.FormatInt(recipeID, 10), fmt.Sprintf("%d%s", ingredientID, uploadedImage.Suffix))
	env.Logger.DebugContext(ctx, "uploading recipe ingredient image")
	imageURL, _, err := env.FileServer.Write(path, uploadedImage.Data)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to upload recipe ingredient image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	err = env.Database.UpdateRecipeIngredientImage(ctx, database.UpdateRecipeIngredientImageParams{
		ImageUrl: pgtype.Text{
			String: imageURL,
			Valid:  true,
		},
		ID: ingredientID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe ingredient image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
