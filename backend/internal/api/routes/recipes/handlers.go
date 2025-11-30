// Package recipes contains handlers for the recipes endpoint.
package recipes

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/fileserver"
	"github.com/matt-dz/wecook/internal/recipe"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	defaultRecipeTitle = "Untitled Recipe"
	maxUploadSize      = 20 << 20 // ~ 20 MB
)

// CreateRecipe godoc
//
//	@Summary	Create a recipe.
//	@Tags		Recipes
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
//	@Tags			Recipes, Ingredients
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
	uploadedImage, err := recipe.ReadImage(r, "image")
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

	// Check recipe ownership
	env.Logger.DebugContext(ctx, "checking recipe ownership")
	recipeID, _ := strconv.ParseInt(string(request.RecipeID), 10, 64)
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: recipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		_ = apiError.EncodeError(w, apiError.RecipeNotOwned, "user does not own recipe", requestID)
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
	if uploadedImage != nil {
		env.Logger.DebugContext(ctx, "uploading image")
		path := fileserver.NewIngredientsImage(strconv.FormatInt(recipeID, 10),
			strconv.FormatInt(ingredientID, 10), uploadedImage.Suffix)
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
	}

	w.WriteHeader(http.StatusCreated)
}

// CreateRecipeStep godoc
//
//	@Summary		Create a recipe step
//	@Description	Creates a new step for a recipe owned by the authenticated user.
//	@Description	Accepts multipart/form-data with required fields and an optional image upload.
//	@Tags			Recipes, Steps
//	@Accept			multipart/form-data
//	@Produce		json
//
//	@Param			recipe-id	formData	string			true	"ID of the recipe the step belongs to"
//	@Param			instruction	formData	string			true	"Step instruction text"
//	@Param			image		formData	file			false	"Optional step image (JPEG/PNG)"
//
//	@Success		201			{string}	string			"Step created"
//	@Failure		400			{object}	apiError.Error	"Bad request / validation error / invalid file type"
//	@Failure		401			{object}	apiError.Error	"Unauthorized"
//	@Failure		403			{object}	apiError.Error	"User does not own recipe"
//	@Failure		500			{object}	apiError.Error	"Internal server error"
//
//	@Security		AccessTokenCookie
//	@Router			/api/recipes/steps [post]
func CreateRecipeStep(w http.ResponseWriter, r *http.Request) {
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
	request := CreateRecipeStepRequest{
		RecipeID:    recipeID(strings.TrimSpace(r.Form.Get("recipe-id"))),
		Instruction: strings.TrimSpace(r.Form.Get("instruction")),
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	uploadedImage, err := recipe.ReadImage(r, "image")
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

	// Check recipe ownership
	env.Logger.DebugContext(ctx, "checking recipe ownership")
	recipeID, _ := strconv.ParseInt(string(request.RecipeID), 10, 64)
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: recipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		_ = apiError.EncodeError(w, apiError.RecipeNotOwned, "user does not own recipe", requestID)
		return
	}

	// Create step
	env.Logger.DebugContext(ctx, "creating step")
	stepID, err := env.Database.CreateRecipeStep(ctx, database.CreateRecipeStepParams{
		RecipeID:    recipeID,
		Instruction: request.Instruction,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create step", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Upload image
	if uploadedImage != nil {
		env.Logger.DebugContext(ctx, "uploading image")
		path := fileserver.NewStepsImage(
			strconv.FormatInt(recipeID, 10), strconv.FormatInt(stepID, 10), uploadedImage.Suffix)
		imageURL, _, err := env.FileServer.Write(path, uploadedImage.Data)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to upload image", slog.Any("error", err))
			_ = apiError.EncodeInternalError(w, requestID)
			return
		}
		err = env.Database.UpdateRecipeStepImage(ctx, database.UpdateRecipeStepImageParams{
			ImageUrl: pgtype.Text{
				String: imageURL,
				Valid:  true,
			},
			ID: stepID,
		})
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to update recipe ingredient image", slog.Any("error", err))
			_ = apiError.EncodeInternalError(w, requestID)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
}

// UpdateRecipeCover godoc
// @Summary      Update a recipe's cover image
// @Description  Replaces the cover image of a recipe owned by the authenticated user.
// @Description  Expects multipart/form-data containing a single image file.
// @Tags         Recipes
// @Accept       multipart/form-data
// @Produce      json
//
// @Param        recipeID   path      string  true   "ID of the recipe to update"
// @Param        image      formData  file    true   "Cover image (JPEG/PNG)"
//
// @Success      201  {string}  string  "Cover image updated"
// @Failure      400  {object}  apiError.Error  "Bad request / invalid image / missing file"
// @Failure      401  {object}  apiError.Error  "Unauthorized"
// @Failure      403  {object}  apiError.Error  "User does not own recipe"
// @Failure      500  {object}  apiError.Error  "Internal server error"
//
// @Security     AccessTokenCookie
// @Router       /api/recipes/{recipeID}/cover [post]
func UpdateRecipeCover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read request
	env.Logger.DebugContext(ctx, "reading request")
	recipeIDQ := recipeID(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)
	uploadedImage, err := recipe.ReadImage(r, "image")
	if errors.Is(err, recipe.ErrNoImageUploaded) {
		env.Logger.ErrorContext(ctx, "no image uploaded")
		_ = apiError.EncodeError(w, apiError.BadRequest, "expected an image in the form", requestID)
		return
	} else if errors.Is(err, recipe.ErrUnsupportedMimeType) {
		env.Logger.ErrorContext(ctx, "unsupported file type", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid file type", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read ingredient image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Check recipe ownership
	env.Logger.DebugContext(ctx, "checking recipe ownershpi")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: recipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		_ = apiError.EncodeError(w, apiError.RecipeNotOwned, "user does not own recipe", requestID)
		return
	}

	// Upload image
	env.Logger.DebugContext(ctx, "uploading image")
	path := fileserver.NewCoverImage(strconv.FormatInt(recipeID, 10), uploadedImage.Suffix)
	location, _, err := env.FileServer.Write(path, uploadedImage.Data)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to write image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	err = env.Database.UpdateRecipeCoverImage(ctx, database.UpdateRecipeCoverImageParams{
		ImageUrl: pgtype.Text{
			String: location,
			Valid:  true,
		},
		ID: recipeID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
