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
//
//	@Summary		Update a recipe's cover image
//	@Description	Replaces the cover image of a recipe owned by the authenticated user.
//	@Description	Expects multipart/form-data containing a single image file.
//	@Tags			Recipes
//	@Accept			multipart/form-data
//	@Produce		json
//
//	@Param			recipeID	path		string			true	"ID of the recipe to update"
//	@Param			image		formData	file			true	"Cover image (JPEG/PNG)"
//
//	@Success		201			{string}	string			"Cover image updated"
//	@Failure		400			{object}	apiError.Error	"Bad request / invalid image / missing file"
//	@Failure		401			{object}	apiError.Error	"Unauthorized"
//	@Failure		403			{object}	apiError.Error	"User does not own recipe"
//	@Failure		500			{object}	apiError.Error	"Internal server error"
//
//	@Security		AccessTokenCookie
//	@Router			/api/recipes/{recipeID}/cover [post]
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

// GetRecipe godoc
//
//	@Summary		Get a recipe and its owner information
//	@Description	Retrieves a recipe by ID, including recipe details and the owner's basic information.
//	@Tags			Recipes
//	@Produce		json
//
//	@Param			recipeID	path		string				true	"ID of the recipe to retrieve"
//
//	@Success		200			{object}	GetRecipeResponse	"Recipe found"
//	@Failure		400			{object}	apiError.Error		"Bad request (invalid recipe ID)"
//	@Failure		404			{object}	apiError.Error		"Recipe not found"
//	@Failure		500			{object}	apiError.Error		"Internal server error"
//
//	@Router			/api/recipes/{recipeID} [get]
func GetRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Read request
	env.Logger.DebugContext(ctx, "reading request")
	recipeIDQ := recipeID(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner")
	row, err := env.Database.GetRecipeAndOwner(ctx, recipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "could not find recipe and owner", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.RecipeNotFound, "recipe not found", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe and owner", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	steps, err := env.Database.GetRecipeSteps(ctx, recipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe steps", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	ingredients, err := env.Database.GetRecipeIngredients(ctx, recipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe ingredients", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	res := GetRecipeResponse{
		Recipe: recipe.RecipeWithIngredientsAndSteps{
			CookeTimeMinutes: uint32(row.CookTimeMinutes.Int32),
			UserID:           row.UserID.Int64,
			CreatedAt:        row.CreatedAt.Time,
			UpdatedAt:        row.UpdatedAt.Time,
			Published:        row.Published,
			Title:            row.Title,
			Description:      row.Description.String,
			Steps:            make([]recipe.RecipeStep, 0),
			Ingredients:      make([]recipe.RecipeIngredient, 0),
		},
		Owner: recipe.RecipeOwner{
			FirstName: row.FirstName,
			LastName:  row.LastName,
			ID:        row.UserID.Int64,
		},
	}
	if row.ImageUrl.String != "" {
		res.Recipe.ImageURL = env.FileServer.FileURL(row.ImageUrl.String)
	}
	for _, step := range steps {
		res.Recipe.Steps = append(res.Recipe.Steps, recipe.RecipeStep{
			ID:          step.ID,
			RecipeID:    step.RecipeID,
			StepNumber:  step.StepNumber,
			Instruction: step.Instruction,
			CreatedAt:   step.CreatedAt.Time,
			UpdatedAt:   step.UpdatedAt.Time,
		})
		if step.ImageUrl.String != "" {
			res.Recipe.Steps[len(res.Recipe.Steps)-1].ImageURL = env.FileServer.FileURL(step.ImageUrl.String)
		}
	}
	for _, ingredient := range ingredients {
		res.Recipe.Ingredients = append(res.Recipe.Ingredients, recipe.RecipeIngredient{
			ID:       ingredient.ID,
			RecipeID: ingredient.RecipeID,
			Quantity: ingredient.Quantity,
			Name:     ingredient.Name,
			Unit:     ingredient.Unit.String,
		})
		if ingredient.ImageUrl.String != "" {
			res.Recipe.Ingredients[len(res.Recipe.Steps)-1].ImageURL = env.FileServer.FileURL(ingredient.ImageUrl.String)
		}
	}
	env.Logger.DebugContext(ctx, "writing response")
	bytes, err := json.Marshal(res)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to marshal response", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// GetPersonalRecipes godoc
// @Summary      Get recipes owned by the authenticated user
// @Description  Returns all recipes created by the authenticated user, including recipe details and owner information.
// @Tags         Recipes
// @Produce      json
//
// @Success      200  {object}  GetPersonalRecipesResponse  "List of personal recipes"
// @Failure      401  {object}  apiError.Error       "Unauthorized"
// @Failure      500  {object}  apiError.Error       "Internal server error"
//
// @Router       /api/recipes/personal [get]
func GetPersonalRecipes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Get recipes
	env.Logger.DebugContext(ctx, "getting user recipes")
	recipes, err := env.Database.GetRecipesByOwner(ctx, userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipes by owner", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.DebugContext(ctx, "writing response")
	resp := GetPersonalRecipesResponse{
		Recipes: make([]recipe.RecipeAndOwner, 0),
	}
	for _, r := range recipes {
		resp.Recipes = append(resp.Recipes, recipe.RecipeAndOwner{
			Recipe: recipe.Recipe{
				UserID:           r.UserID.Int64,
				Published:        r.Published,
				CookeTimeMinutes: uint32(r.CookTimeMinutes.Int32),
				CreatedAt:        r.CreatedAt.Time,
				UpdatedAt:        r.UpdatedAt.Time,
				Title:            r.Title,
				Description:      r.Description.String,
			},
			Owner: recipe.RecipeOwner{
				ID:        r.UserID.Int64,
				FirstName: r.FirstName,
				LastName:  r.LastName,
			},
		})
		if r.ImageUrl.String != "" {
			resp.Recipes[len(resp.Recipes)-1].Recipe.ImageURL = env.FileServer.FileURL(r.ImageUrl.String)
		}
	}
	marshaled, err := json.Marshal(resp)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to marshal response", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(marshaled)
}
