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
//	@Summary		Create a new recipe
//	@Description	Creates a new (empty) recipe for the authenticated user.
//	@Tags			Recipes
//	@Accept			json
//	@Produce		json
//
//	@Success		200	{object}	CreateRecipeResponse	"Recipe successfully created"
//	@Failure		400	{object}	apiError.Error			"Bad request"
//	@Failure		401	{object}	apiError.Error			"Unauthorized — missing or invalid access token cookie"
//	@Failure		500	{object}	apiError.Error			"Internal server error"
//
//	@Router			/api/recipes [post]
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
		RecipeID: integer64(strings.TrimSpace(r.Form.Get("recipe-id"))),
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
			Valid:  request.Unit != "",
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
		RecipeID:    integer64(strings.TrimSpace(r.Form.Get("recipe-id"))),
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
		env.Logger.ErrorContext(ctx, "failed to read step image", slog.Any("error", err))
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
	recipeIDQ := integer64(chi.URLParam(r, "recipeID"))
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
	recipeIDQ := integer64(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner")
	row, err := env.Database.GetPublishedRecipeAndOwner(ctx, recipeID)
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
			CookTimeAmount: row.CookTimeAmount.Int32,
			CookTimeUnit:   string(row.CookTimeUnit.TimeUnit),
			PrepTimeAmount: row.PrepTimeAmount.Int32,
			PrepTimeUnit:   string(row.PrepTimeUnit.TimeUnit),
			UserID:         row.UserID.Int64,
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
			Published:      row.Published,
			Title:          row.Title,
			ID:             row.ID,
			Servings:       row.Servings.Float32,
			Description:    row.Description.String,
			Steps:          make([]recipe.RecipeStep, 0),
			Ingredients:    make([]recipe.RecipeIngredient, 0),
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
			res.Recipe.Ingredients[len(res.Recipe.Ingredients)-1].ImageURL = env.FileServer.FileURL(ingredient.ImageUrl.String)
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
//
//	@Summary		Get recipes owned by the authenticated user
//	@Description	Returns all recipes created by the authenticated user, including recipe details and owner information.
//	@Tags			Recipes
//	@Produce		json
//
//	@Success		200	{object}	GetPersonalRecipesResponse	"List of personal recipes"
//	@Failure		401	{object}	apiError.Error				"Unauthorized"
//	@Failure		500	{object}	apiError.Error				"Internal server error"
//
//	@Router			/api/recipes/personal [get]
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
				UserID:         r.UserID.Int64,
				Published:      r.Published,
				CookTimeAmount: r.CookTimeAmount.Int32,
				CookTimeUnit:   string(r.CookTimeUnit.TimeUnit),
				PrepTimeAmount: r.PrepTimeAmount.Int32,
				PrepTimeUnit:   string(r.PrepTimeUnit.TimeUnit),
				CreatedAt:      r.CreatedAt.Time,
				UpdatedAt:      r.UpdatedAt.Time,
				Title:          r.Title,
				Description:    r.Description.String,
				ID:             r.ID,
				Servings:       r.Servings.Float32,
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

// DeleteRecipe godoc
//
//	@Summary		Delete a recipe
//	@Description	Deletes a recipe owned by the authenticated user.
//	@Description	This operation is idempotent — deleting an already deleted or non-owned recipe results in a 404.
//	@Tags			Recipes
//	@Produce		json
//
//	@Param			recipeID	path		string			true	"ID of the recipe to delete"
//
//	@Success		204			{string}	string			"Recipe deleted successfully"
//	@Failure		400			{object}	apiError.Error	"Bad request (invalid recipe ID)"
//	@Failure		401			{object}	apiError.Error	"Unauthorized"
//	@Failure		404			{object}	apiError.Error	"Recipe not found or not owned by user"
//	@Failure		500			{object}	apiError.Error	"Internal server error"
//
//	@Security		AccessTokenCookie
//	@Router			/api/recipes/{recipeID} [delete]
func DeleteRecipe(w http.ResponseWriter, r *http.Request) {
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
	recipeIDQ := integer64(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)

	// Check ownership & existence
	env.Logger.DebugContext(ctx, "checking user ownership")
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
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}

	// Delete recipe
	env.Logger.DebugContext(ctx, "deleting recipe")
	if err := env.Database.DeleteRecipe(ctx, recipeID); err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete recipe", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteRecipeIngredient godoc
//
//	@Summary		Delete a recipe ingredient
//	@Description	Deletes an ingredient from a recipe owned by the authenticated user.
//	@Description	This operation is idempotent — deleting a non-existent ingredient returns 404.
//	@Tags			Recipes, Ingredients
//	@Produce		json
//
//	@Param			recipeID		path		string			true	"ID of the recipe"
//	@Param			ingredientID	path		string			true	"ID of the ingredient to delete"
//
//	@Success		204				{string}	string			"Ingredient deleted successfully"
//	@Failure		400				{object}	apiError.Error	"Invalid recipe ID or ingredient ID"
//	@Failure		401				{object}	apiError.Error	"Unauthorized"
//	@Failure		404				{object}	apiError.Error	"Recipe not found, not owned by user, or ingredient not found"
//	@Failure		500				{object}	apiError.Error	"Internal server error"
//
//	@Router			/api/recipes/{recipeID}/ingredients/{ingredientID} [delete]
func DeleteRecipeIngredient(w http.ResponseWriter, r *http.Request) {
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
	recipeIDQ := integer64(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid recipe id", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)
	ingredientIDQ := integer64(chi.URLParam(r, "ingredientID"))
	if err := ingredientIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate ingredient id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid ingredient id", requestID)
		return
	}
	ingredientID, _ := strconv.ParseInt(string(ingredientIDQ), 10, 64)

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
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
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}
	env.Logger.DebugContext(ctx, "checking ingredient existence")
	exists, err := env.Database.GetRecipeIngredientExistence(ctx, ingredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get ingredient existence", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !exists {
		env.Logger.ErrorContext(ctx, "ingredient not found", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.IngredientNotFound, "ingredient not found", requestID)
		return
	}

	// Get ingredient image url
	env.Logger.DebugContext(ctx, "getting ingredient image url")
	imageURL, err := env.Database.GetRecipeIngredientImageURL(ctx, ingredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get ingredient image url", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Delete ingredient
	env.Logger.DebugContext(ctx, "deleting ingredient")
	err = env.Database.DeleteRecipeIngredient(ctx, ingredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete ingredient", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if imageURL.String != "" {
		err := env.FileServer.Delete(imageURL.String)
		if err != nil {
			env.Logger.WarnContext(ctx, "failed to delete image, manual cleanup required",
				slog.Any("error", err),
				slog.String("image-path", imageURL.String))
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteRecipeStep godoc
//
//	@Summary		Delete a recipe step
//	@Description	Deletes a specific step from a recipe.
//	@Description	This operation is **idempotent**—deleting the same step twice will return `StepNotFound`.
//	@Tags			Recipes, Steps
//	@Security		AccessToken
//	@Param			recipeID	path	int	true	"ID of the recipe"
//	@Param			stepID		path	int	true	"ID of the step to delete"
//	@Success		204			"Step successfully deleted"
//	@Failure		400			{object}	apiError.Error	"Invalid recipe ID or step ID"
//	@Failure		401			{object}	apiError.Error	"Unauthorized"
//	@Failure		404			{object}	apiError.Error	"Recipe not found, user does not own the recipe, or step not found"
//	@Failure		500			{object}	apiError.Error	"Internal server error"
//	@Router			/api/recipes/{recipeID}/steps/{stepID} [delete]
func DeleteRecipeStep(w http.ResponseWriter, r *http.Request) {
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
	recipeIDQ := integer64(chi.URLParam(r, "recipeID"))
	if err := recipeIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate recipe id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid recipe id", requestID)
		return
	}
	recipeID, _ := strconv.ParseInt(string(recipeIDQ), 10, 64)
	stepIDQ := integer64(chi.URLParam(r, "stepID"))
	if err := stepIDQ.Validate(); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate step id", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid step id", requestID)
		return
	}
	stepID, _ := strconv.ParseInt(string(stepIDQ), 10, 64)

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
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
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}
	env.Logger.DebugContext(ctx, "checking step existence")
	exists, err := env.Database.GetRecipeStepExistence(ctx, stepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step existence", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !exists {
		env.Logger.ErrorContext(ctx, "step not found", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.StepNotFound, "step not found", requestID)
		return
	}

	// Get step image url
	env.Logger.DebugContext(ctx, "getting step image url")
	imageURL, err := env.Database.GetRecipeStepImageURL(ctx, stepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get step image url", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Delete step
	env.Logger.DebugContext(ctx, "deleting step")
	err = env.Database.DeleteRecipeStep(ctx, stepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete step", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if imageURL.String != "" {
		err := env.FileServer.Delete(imageURL.String)
		if err != nil {
			env.Logger.WarnContext(ctx, "failed to delete image, manual cleanup required",
				slog.Any("error", err),
				slog.String("image-path", imageURL.String))
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateRecipeStep godoc
//
//	@Summary		Update a recipe step
//	@Description	Updates a specific step within a recipe.
//	@Description	Supports partial updates: users may update the instruction, the image, or both.
//	@Tags			Recipes, Steps
//	@Security		AccessToken
//	@Param			recipeID	path		int		true	"ID of the recipe"
//	@Param			stepID		path		int		true	"ID of the step to update"
//	@Param			instruction	formData	string	false	"Updated instruction text"
//	@Param			image		formData	file	false	"Updated step image (optional)"
//	@Accept			multipart/form-data
//	@Produce		json
//	@Success		204	"Step updated successfully"
//	@Failure		400	{object}	apiError.Error	"Bad request — invalid IDs, invalid file type, or malformed form"
//	@Failure		401	{object}	apiError.Error	"Unauthorized"
//	@Failure		404	{object}	apiError.Error	"Recipe not found, not owned by user, or step not found"
//	@Failure		500	{object}	apiError.Error	"Internal server error"
//	@Router			/api/recipes/{recipeID}/steps/{stepID} [patch]
func UpdateRecipeStep(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read recipeID and stepID
	env.Logger.DebugContext(ctx, "reading request")
	request := UpdateRecipeStepRequest{
		RecipeID: integer64(chi.URLParam(r, "recipeID")),
		StepID:   integer64(chi.URLParam(r, "stepID")),
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := request.RecipeID.Int()
	stepID, _ := request.StepID.Int()

	// Check ownership and existence
	env.Logger.DebugContext(ctx, "checking user ownership")
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
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}
	env.Logger.DebugContext(ctx, "checking step existence")
	exists, err := env.Database.GetRecipeStepExistence(ctx, stepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step existence", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !exists {
		env.Logger.ErrorContext(ctx, "step not found", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.StepNotFound, "step not found", requestID)
		return
	}

	// Parse form
	env.Logger.DebugContext(ctx, "parsing form")
	if r.ContentLength == 0 {
		env.Logger.DebugContext(ctx, "form is empty")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		env.Logger.ErrorContext(ctx, "failed to parse multipart form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid multipart form", requestID)
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
		env.Logger.ErrorContext(ctx, "failed to read step image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Update image
	params := database.UpdateRecipeStepParams{
		ID: stepID,
	}
	if uploadedImage != nil {
		env.Logger.DebugContext(ctx, "writing new recipe image")
		location, _, err := env.FileServer.Write(fileserver.NewStepsImage(request.RecipeID.String(),
			request.StepID.String(), uploadedImage.Suffix), uploadedImage.Data)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to write new image", slog.Any("error", err))
			_ = apiError.EncodeInternalError(w, requestID)
			return
		}
		params.ImageUrl.String = location
		params.ImageUrl.Valid = true
	}

	// Update recipe step
	env.Logger.DebugContext(ctx, "updating recipe step")
	if r.Form.Has("instruction") {
		env.Logger.DebugContext(ctx, "updating recipe instruction")
		params.Instruction.String = r.Form.Get("instruction")
		params.Instruction.Valid = true
	}
	err = env.Database.UpdateRecipeStep(ctx, params)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe step", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateRecipeIngredient godoc
//
//	@Summary		Update a recipe ingredient
//	@Description	Partially updates a recipe ingredient.
//	@Description	Supports updating the ingredient's name, unit, quantity, or image.
//	@Description	Fields are only updated if provided. Sending an empty body results in a no-op.
//	@Tags			Recipes, Ingredients
//	@Security		AccessToken
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			recipeID		path		int		true	"ID of the recipe"
//	@Param			ingredientID	path		int		true	"ID of the ingredient to update"
//	@Param			quantity		formData	string	false	"Updated quantity (float)"
//	@Param			unit			formData	string	false	"Updated unit (e.g. 'tsp', 'grams')"
//	@Param			name			formData	string	false	"Updated ingredient name"
//	@Param			image			formData	file	false	"Updated ingredient image"
//	@Success		204				"Ingredient updated successfully"
//	@Failure		400				{object}	apiError.Error	"Bad request (invalid form or invalid fields)"
//	@Failure		401				{object}	apiError.Error	"Unauthorized"
//	@Failure		404				{object}	apiError.Error	"Recipe not found, ingredient not found, or recipe not owned by user"
//	@Failure		500				{object}	apiError.Error	"Internal server error"
//	@Router			/api/recipes/{recipeID}/ingredients/{ingredientID} [patch]
func UpdateRecipeIngredient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read recipeID and stepID
	env.Logger.DebugContext(ctx, "reading request")
	request := UpdateRecipeIngredientRequest{
		RecipeID:     integer64(chi.URLParam(r, "recipeID")),
		IngredientID: integer64(chi.URLParam(r, "ingredientID")),
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := request.RecipeID.Int()
	ingredientID, _ := request.IngredientID.Int()

	// Check ownership and existence
	env.Logger.DebugContext(ctx, "checking user ownership")
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
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}
	env.Logger.DebugContext(ctx, "checking ingredient existence")
	exists, err := env.Database.GetRecipeIngredientExistence(ctx, ingredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check ingredient existence", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if !exists {
		env.Logger.ErrorContext(ctx, "ingredient not found", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.IngredientNotFound, "ingredient not found", requestID)
		return
	}

	// Parse form
	env.Logger.DebugContext(ctx, "parsing form")
	if r.ContentLength == 0 {
		env.Logger.DebugContext(ctx, "form is empty")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		env.Logger.ErrorContext(ctx, "failed to parse multipart form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid multipart form", requestID)
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
		env.Logger.ErrorContext(ctx, "failed to read step image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	form := UpdateRecipeIngredientForm{
		Quantity: r.Form.Get("quantity"),
		Unit:     r.Form.Get("unit"),
		Name:     r.Form.Get("name"),
	}
	if err := validate.Struct(form); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "failed to validate form", requestID)
		return
	}

	// Update ingredient
	env.Logger.DebugContext(ctx, "updating ingredient")
	params := database.UpdateRecipeIngredientParams{
		ID: ingredientID,
	}
	if uploadedImage != nil {
		location, _, err := env.FileServer.Write(fileserver.NewIngredientsImage(
			request.RecipeID.String(), request.IngredientID.String(), uploadedImage.Suffix), uploadedImage.Data)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to write image", slog.Any("error", err))
			_ = apiError.EncodeInternalError(w, requestID)
			return
		}
		params.ImageUrl.String = location
		params.ImageUrl.Valid = true
	}
	if r.Form.Has("quantity") {
		quantity, err := strconv.ParseFloat(form.Quantity, 32)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse float", slog.Any("error", err))
			_ = apiError.EncodeError(w, apiError.BadRequest, "invalid quantity", requestID)
			return
		}
		params.Quantity.Float32 = float32(quantity)
		params.Quantity.Valid = true
	}
	if r.Form.Has("unit") {
		params.Unit.String = form.Unit
		params.Unit.Valid = true
	}
	if r.Form.Has("name") {
		params.Name.String = form.Name
		params.Name.Valid = true
	}
	err = env.Database.UpdateRecipeIngredient(ctx, params)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe ingredient", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateRecipe godoc
//
//	@Summary		Update a recipe
//	@Description	Partially updates a recipe. Supports updating title, description,
//	@Description	published status, cook time, prep time, servings, and cover image.
//	@Description	Fields not provided are left unchanged. If the request body is empty,
//	@Description	this is treated as a no-op and returns 204.
//	@Tags			Recipes
//	@Security		AccessToken
//	@Accept			multipart/form-data
//	@Produce		json
//
//	@Param			recipeID			path		int		true	"ID of the recipe"
//	@Param			title				formData	string	false	"New title"
//	@Param			description			formData	string	false	"New description"
//	@Param			published			formData	bool	false	"Published status"
//	@Param			cook-time-amount	formData	int		false	"Cook time amount"
//	@Param			cook-time-unit		formData	string	false	"Cook time unit (minutes, hours, days)"
//	@Param			prep-time-amount	formData	int		false	"Prep time amount"
//	@Param			prep-time-unit		formData	string	false	"Prep time unit (minutes, hours, days)"
//	@Param			servings			formData	number	false	"Servings"
//	@Param			image				formData	file	false	"New cover image"
//
//	@Success		204					"Recipe updated successfully"
//	@Failure		400					{object}	apiError.Error	"Bad request (invalid form or data)"
//	@Failure		401					{object}	apiError.Error	"Unauthorized"
//	@Failure		404					{object}	apiError.Error	"Recipe not found or user does not own it"
//	@Failure		500					{object}	apiError.Error	"Internal server error"
//
//	@Router			/api/recipes/{recipeID} [patch]
func UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read recipeID
	env.Logger.DebugContext(ctx, "reading request")
	request := UpdateRecipeRequest{
		RecipeID: integer64(chi.URLParam(r, "recipeID")),
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := request.RecipeID.Int()

	// Check ownership
	env.Logger.DebugContext(ctx, "checking ownership")
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
		env.Logger.ErrorContext(ctx, "user does not own recipe", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.RecipeNotFound,
			"recipe does not exist or user does not own it", requestID)
		return
	}

	// Parse form
	env.Logger.DebugContext(ctx, "parsing form")
	if r.ContentLength == 0 {
		env.Logger.DebugContext(ctx, "form is empty")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		env.Logger.ErrorContext(ctx, "failed to parse multipart form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid multipart form", requestID)
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
		env.Logger.ErrorContext(ctx, "failed to read step image", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	form := UpdateRecipeForm{
		Title:          r.Form.Get("title"),
		Description:    r.Form.Get("description"),
		Published:      r.Form.Get("published"),
		CookTimeAmount: integer32(r.Form.Get("cook-time-amount")),
		CookTimeUnit:   r.Form.Get("cook-time-unit"),
		PrepTimeAmount: integer32(r.Form.Get("prep-time-amount")),
		PrepTimeUnit:   r.Form.Get("prep-time-unit"),
		Servings:       r.Form.Get("servings"),
	}
	if err := validate.Struct(form); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate form", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "failed to validate form", requestID)
		return
	}

	// Update recipe
	updateParams := database.UpdateRecipeParams{
		ID: recipeID,
	}
	if uploadedImage != nil {
		env.Logger.DebugContext(ctx, "updating image")
		location, _, err := env.FileServer.Write(
			fileserver.NewCoverImage(request.RecipeID.String(), uploadedImage.Suffix),
			uploadedImage.Data,
		)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to update image", slog.Any("error", err))
			_ = apiError.EncodeInternalError(w, requestID)
			return
		}
		updateParams.ImageUrl.String = location
		updateParams.ImageUrl.Valid = true
	}
	if r.Form.Has("title") {
		updateParams.Title.String = form.Title
		updateParams.Title.Valid = true
	}
	if r.Form.Has("description") {
		updateParams.Description.String = form.Description
		updateParams.Description.Valid = true
	}
	if r.Form.Has("published") {
		published, err := strconv.ParseBool(form.Published)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse published field", slog.Any("error", err))
			_ = apiError.EncodeError(w, apiError.BadRequest, "invalid published", requestID)
			return
		}
		updateParams.Published.Bool = published
		updateParams.Published.Valid = true
	}
	if r.Form.Has("cook-time-amount") {
		cookTimeAmount, err := form.CookTimeAmount.Int()
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse cook-time-amount field", slog.Any("error", err))
			_ = apiError.EncodeError(w, apiError.BadRequest, "invalid cook time amount", requestID)
			return
		}
		updateParams.CookTimeAmount.Int32 = int32(cookTimeAmount)
		updateParams.CookTimeAmount.Valid = true
	}
	if r.Form.Has("cook-time-unit") {
		updateParams.CookTimeUnit.TimeUnit = database.TimeUnit(form.CookTimeUnit)
		updateParams.CookTimeUnit.Valid = true
	}
	if r.Form.Has("prep-time-amount") {
		prepTimeAmount, err := form.PrepTimeAmount.Int()
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse prep-time-amount field", slog.Any("error", err))
			_ = apiError.EncodeError(w, apiError.BadRequest, "invalid prep time amount", requestID)
			return
		}
		updateParams.PrepTimeAmount.Int32 = int32(prepTimeAmount)
		updateParams.PrepTimeAmount.Valid = true
	}
	if r.Form.Has("prep-time-unit") {
		updateParams.PrepTimeUnit.TimeUnit = database.TimeUnit(form.PrepTimeUnit)
		updateParams.PrepTimeUnit.Valid = true
	}
	if r.Form.Has("servings") {
		servings, err := strconv.ParseFloat(form.Servings, 32)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to parse servings field", slog.Any("error", err))
			_ = apiError.EncodeError(w, apiError.BadRequest, "invalid servings", requestID)
			return
		}
		updateParams.Servings.Float32 = float32(servings)
		updateParams.Servings.Valid = true

	}
	err = env.Database.UpdateRecipe(ctx, updateParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPersonalRecipe godoc
//
//	@Summary		Get a personal (owned) recipe
//	@Description	Retrieves a recipe **only if it is owned by the authenticated user**.
//	@Description	Includes full recipe details: steps, ingredients, metadata, and owner info.
//	@Tags			Recipes
//	@Security		AccessToken
//	@Param			recipeID	path	int	true	"ID of the recipe"
//	@Produce		json
//	@Success		200	{object}	GetRecipeResponse	"Full recipe with steps and ingredients"
//	@Failure		400	{object}	apiError.Error		"Bad request — invalid recipe ID"
//	@Failure		404	{object}	apiError.Error		"Recipe not found or not owned by user"
//	@Failure		500	{object}	apiError.Error		"Internal server error"
//	@Router			/api/recipes/personal/{recipeID} [get]
func GetPersonalRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user id", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Read request
	env.Logger.DebugContext(ctx, "reading request")
	request := GetRecipeRequest{
		RecipeID: integer64(chi.URLParam(r, "recipeID")),
	}
	validate := validator.New()
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "failed to validate request", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "bad request", requestID)
		return
	}
	recipeID, _ := request.RecipeID.Int()

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner", slog.Int64("recipe-id", recipeID))
	row, err := env.Database.GetRecipeAndOwner(ctx, recipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "could not find recipe and owner", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.RecipeNotFound, "recipe not owned by user or does not exist", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe and owner", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if row.UserID.Int64 != userID {
		env.Logger.ErrorContext(ctx, "not owned by user")
		_ = apiError.EncodeError(w, apiError.RecipeNotFound, "recipe not owned by user or does not exist", requestID)
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
			CookTimeAmount: row.CookTimeAmount.Int32,
			CookTimeUnit:   string(row.CookTimeUnit.TimeUnit),
			PrepTimeAmount: row.PrepTimeAmount.Int32,
			PrepTimeUnit:   string(row.PrepTimeUnit.TimeUnit),
			UserID:         row.UserID.Int64,
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
			Published:      row.Published,
			Title:          row.Title,
			Description:    row.Description.String,
			ID:             row.ID,
			Servings:       row.Servings.Float32,
			Steps:          make([]recipe.RecipeStep, 0),
			Ingredients:    make([]recipe.RecipeIngredient, 0),
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
			res.Recipe.Ingredients[len(res.Recipe.Ingredients)-1].ImageURL = env.FileServer.FileURL(ingredient.ImageUrl.String)
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
