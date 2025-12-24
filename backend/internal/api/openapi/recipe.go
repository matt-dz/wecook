package client

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/fileserver"
	"github.com/matt-dz/wecook/internal/form"
)

const (
	defaultRecipeTitle = "Untitled Recipe"
	maxUploadSize      = 20 << 20 // ~ 20 MB
)

// buildRecipeWithIngredientsAndSteps is a helper function that fetches recipe details
// (steps and ingredients) and builds the response structure.
func buildRecipeWithIngredientsAndSteps(
	ctx context.Context,
	env *env.Env,
	recipeID int64,
	row database.GetRecipeAndOwnerRow,
) (RecipeWithIngredientsAndSteps, RecipeOwner, error) {
	// Get steps
	steps, err := env.Database.GetRecipeSteps(ctx, recipeID)
	if err != nil {
		return RecipeWithIngredientsAndSteps{}, RecipeOwner{}, err
	}

	// Get ingredients
	ingredients, err := env.Database.GetRecipeIngredients(ctx, recipeID)
	if err != nil {
		return RecipeWithIngredientsAndSteps{}, RecipeOwner{}, err
	}

	// Build owner
	owner := RecipeOwner{
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Id:        row.ID,
	}

	// Build recipe
	recipe := RecipeWithIngredientsAndSteps{
		UserId:      row.UserID.Int64,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		Published:   row.Published,
		Title:       row.Title,
		Id:          row.ID,
		Steps:       make([]RecipeStep, 0),
		Ingredients: make([]RecipeIngredient, 0),
	}
	if row.Servings.Valid {
		servings := row.Servings.Float32
		recipe.Servings = &servings
	}
	if row.Description.Valid {
		description := row.Description.String
		recipe.Description = &description
	}
	if row.CookTimeAmount.Valid {
		cookTime := row.CookTimeAmount.Int32
		recipe.CookTimeAmount = &cookTime
	}
	if row.CookTimeUnit.Valid {
		cookTimeUnit := TimeUnit(row.CookTimeUnit.TimeUnit)
		recipe.CookTimeUnit = &cookTimeUnit
	}
	if row.PrepTimeAmount.Valid {
		prepTime := row.PrepTimeAmount.Int32
		recipe.PrepTimeAmount = &prepTime
	}
	if row.PrepTimeUnit.Valid {
		prepTimeUnit := TimeUnit(row.PrepTimeUnit.TimeUnit)
		recipe.PrepTimeUnit = &prepTimeUnit
	}

	// Add recipe image URL if exists
	if row.ImageUrl.String != "" {
		imageURL := env.FileStore.FileURL(row.ImageUrl.String)
		recipe.ImageUrl = &imageURL
	}

	// Build steps
	for _, step := range steps {
		newStep := RecipeStep{
			Id:         step.ID,
			RecipeId:   step.RecipeID,
			StepNumber: step.StepNumber,
		}
		if step.Instruction.Valid {
			instr := step.Instruction.String
			newStep.Instruction = &instr
		}
		if step.ImageUrl.Valid {
			imageURL := env.FileStore.FileURL(step.ImageUrl.String)
			newStep.ImageUrl = &imageURL
		}
		recipe.Steps = append(recipe.Steps, newStep)
	}

	// Build ingredients
	for _, ingredient := range ingredients {
		newIngredient := RecipeIngredient{
			Id:       ingredient.ID,
			RecipeId: ingredient.RecipeID,
		}
		if ingredient.Quantity.Valid {
			quantity := ingredient.Quantity.Float32
			newIngredient.Quantity = &quantity
		}
		if ingredient.Name.Valid {
			name := ingredient.Name.String
			newIngredient.Name = &name
		}
		if ingredient.Unit.Valid {
			unit := ingredient.Unit.String
			newIngredient.Unit = &unit
		}
		if ingredient.ImageUrl.Valid {
			imageURL := env.FileStore.FileURL(ingredient.ImageUrl.String)
			newIngredient.ImageUrl = &imageURL
		}
		recipe.Ingredients = append(recipe.Ingredients, newIngredient)
	}

	return recipe, owner, nil
}

func (Server) PostApiRecipes(ctx context.Context,
	request PostApiRecipesRequestObject,
) (PostApiRecipesResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipes500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
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
		return PostApiRecipes500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiRecipes201JSONResponse{
		RecipeId: recipeID,
	}, nil
}

func (Server) GetApiRecipesRecipeIDPublic(ctx context.Context,
	request GetApiRecipesRecipeIDPublicRequestObject,
) (GetApiRecipesRecipeIDPublicResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner")
	publishedRow, err := env.Database.GetPublishedRecipeAndOwner(ctx, request.RecipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "recipe does not exist", slog.Any("error", err))
		return GetApiRecipesRecipeIDPublic404JSONResponse{
			Code:    apiError.RecipeNotFound.String(),
			Status:  apiError.RecipeNotFound.StatusCode(),
			Message: "recipe does not exist or is not public",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe and owner", slog.Any("error", err))
		return GetApiRecipesRecipeIDPublic500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Convert to GetRecipeAndOwnerRow (same structure, different field order)
	row := database.GetRecipeAndOwnerRow{
		UserID:         publishedRow.UserID,
		ImageUrl:       publishedRow.ImageUrl,
		Title:          publishedRow.Title,
		Description:    publishedRow.Description,
		CreatedAt:      publishedRow.CreatedAt,
		UpdatedAt:      publishedRow.UpdatedAt,
		Published:      publishedRow.Published,
		ID:             publishedRow.ID,
		CookTimeAmount: publishedRow.CookTimeAmount,
		CookTimeUnit:   publishedRow.CookTimeUnit,
		PrepTimeAmount: publishedRow.PrepTimeAmount,
		PrepTimeUnit:   publishedRow.PrepTimeUnit,
		Servings:       publishedRow.Servings,
		FirstName:      publishedRow.FirstName,
		LastName:       publishedRow.LastName,
		ID_2:           publishedRow.ID_2,
	}

	// Build recipe with ingredients and steps
	recipe, owner, err := buildRecipeWithIngredientsAndSteps(ctx, env, request.RecipeID, row)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to build recipe details", slog.Any("error", err))
		return GetApiRecipesRecipeIDPublic500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return GetApiRecipesRecipeIDPublic200JSONResponse{
		Owner:  owner,
		Recipe: recipe,
	}, nil
}

func (Server) GetApiRecipesRecipeID(ctx context.Context,
	request GetApiRecipesRecipeIDRequestObject) (
	GetApiRecipesRecipeIDResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return GetApiRecipesRecipeID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return GetApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		return GetApiRecipesRecipeID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe does not exist or user does not own it",
			ErrorId: requestID,
		}, nil
	}

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner")
	row, err := env.Database.GetRecipeAndOwner(ctx, request.RecipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "recipe does not exist", slog.Any("error", err))
		return GetApiRecipesRecipeID404JSONResponse{
			Code:    apiError.RecipeNotFound.String(),
			Status:  apiError.RecipeNotFound.StatusCode(),
			Message: "recipe does not exist",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe and owner", slog.Any("error", err))
		return GetApiRecipesRecipeID500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Build recipe with ingredients and steps
	recipe, owner, err := buildRecipeWithIngredientsAndSteps(ctx, env, request.RecipeID, row)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to build recipe details", slog.Any("error", err))
		return GetApiRecipesRecipeID500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return GetApiRecipesRecipeID200JSONResponse{
		Owner:  owner,
		Recipe: recipe,
	}, nil
}

func (Server) DeleteApiRecipesRecipeID(
	ctx context.Context,
	request DeleteApiRecipesRecipeIDRequestObject,
) (DeleteApiRecipesRecipeIDResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership & existence
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		return DeleteApiRecipesRecipeID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe does not exist or user does not own it",
			ErrorId: requestID,
		}, nil
	}

	// Get recipe to retrieve image URLs
	env.Logger.DebugContext(ctx, "getting recipe details")
	recipe, err := env.Database.GetRecipeAndOwner(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Get all steps to retrieve their image URLs
	env.Logger.DebugContext(ctx, "getting recipe steps")
	steps, err := env.Database.GetRecipeSteps(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe steps", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Get all ingredients to retrieve their image URLs
	env.Logger.DebugContext(ctx, "getting recipe ingredients")
	ingredients, err := env.Database.GetRecipeIngredients(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe ingredients", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Delete recipe image from file server
	if recipe.ImageUrl.Valid && recipe.ImageUrl.String != "" {
		env.Logger.DebugContext(ctx, "deleting recipe image", slog.String("path", recipe.ImageUrl.String))
		if err := env.FileStore.DeleteURLPath(recipe.ImageUrl.String); err != nil {
			env.Logger.WarnContext(ctx, "failed to delete recipe image", slog.Any("error", err))
		}
	}

	// Delete step images from file server
	for _, step := range steps {
		if step.ImageUrl.Valid && step.ImageUrl.String != "" {
			env.Logger.DebugContext(ctx, "deleting step image", slog.String("path", step.ImageUrl.String))
			if err := env.FileStore.DeleteURLPath(step.ImageUrl.String); err != nil {
				env.Logger.WarnContext(ctx, "failed to delete step image", slog.Any("error", err))
			}
		}
	}

	// Delete ingredient images from file server
	for _, ingredient := range ingredients {
		if ingredient.ImageUrl.Valid && ingredient.ImageUrl.String != "" {
			env.Logger.DebugContext(ctx, "deleting ingredient image", slog.String("path", ingredient.ImageUrl.String))
			if err := env.FileStore.DeleteURLPath(ingredient.ImageUrl.String); err != nil {
				env.Logger.WarnContext(ctx, "failed to delete ingredient image", slog.Any("error", err))
			}
		}
	}

	// Delete recipe from database
	env.Logger.DebugContext(ctx, "deleting recipe from database")
	if err := env.Database.DeleteRecipe(ctx, request.RecipeID); err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete recipe", slog.Any("error", err))
		return DeleteApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeID204Response{}, nil
}

func (Server) PostApiRecipesRecipeIDIngredients(ctx context.Context,
	request PostApiRecipesRecipeIDIngredientsRequestObject,
) (PostApiRecipesRecipeIDIngredientsResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredients500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredients500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		return PostApiRecipesRecipeIDIngredients500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	env.Logger.DebugContext(ctx, "creating ingredient")
	row, err := env.Database.CreateEmptyRecipeIngredient(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create ingredient", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredients500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiRecipesRecipeIDIngredients200JSONResponse{
		Id: row.ID,
	}, nil
}

func (Server) PatchApiRecipesRecipeIDIngredientsIngredientID(ctx context.Context,
	request PatchApiRecipesRecipeIDIngredientsIngredientIDRequestObject,
) (PatchApiRecipesRecipeIDIngredientsIngredientIDResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PatchApiRecipesRecipeIDIngredientsIngredientID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsIngredient, err := env.Database.CheckIngredientOwnership(ctx, database.CheckIngredientOwnershipParams{
		RecipeID:     request.RecipeID,
		IngredientID: request.IngredientID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return PatchApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsIngredient {
		env.Logger.ErrorContext(ctx, "user does not own recipe or ingredient")
		return PatchApiRecipesRecipeIDIngredientsIngredientID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/ingredient does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	env.Logger.DebugContext(ctx, "updating ingredient")
	updateParams := database.UpdateRecipeIngredientParams{
		ID: request.IngredientID,
	}
	if request.Body.Name != nil {
		updateParams.Name.String = *request.Body.Name
		updateParams.Name.Valid = true
	}
	if request.Body.Quantity != nil {
		updateParams.Quantity.Float32 = *request.Body.Quantity
		updateParams.Quantity.Valid = true
	}
	if request.Body.Unit != nil {
		updateParams.Unit.String = *request.Body.Unit
		updateParams.Unit.Valid = true
	}
	row, err := env.Database.UpdateRecipeIngredient(ctx, updateParams)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "ingredient does not exist", slog.Any("error", err))
		return PatchApiRecipesRecipeIDIngredientsIngredientID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/ingredient does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update ingredient", slog.Any("error", err))
		return PatchApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	res := PatchApiRecipesRecipeIDIngredientsIngredientID200JSONResponse{
		Id:   row.ID,
		Name: row.Name.String,
	}
	if row.Quantity.Valid {
		res.Quantity = &row.Quantity.Float32
	}
	if row.Unit.Valid {
		res.Unit = &row.Unit.String
	}
	if row.ImageUrl.Valid {
		imageURL := env.FileStore.FileURL(row.ImageUrl.String)
		res.ImageUrl = &imageURL
	}
	return res, nil
}

func (Server) PostApiRecipesRecipeIDIngredientsIngredientIDImage(ctx context.Context,
	request PostApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject,
) (PostApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsRecipe, err := env.Database.CheckIngredientOwnership(ctx, database.CheckIngredientOwnershipParams{
		RecipeID:     request.RecipeID,
		IngredientID: request.IngredientID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		env.Logger.ErrorContext(ctx, "user does not own recipe or ingredient")
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/ingredient does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Read image
	env.Logger.DebugContext(ctx, "reading recipe image")
	requestForm, err := request.Body.ReadForm(form.MaximumUploadSize)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read form", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid form",
			ErrorId: requestID,
		}, nil
	}
	imageFile, err := requestForm.File["image"][0].Open()
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to open image", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}
	defer func() { _ = imageFile.Close() }()
	file, err := form.ReadFile(imageFile)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read image", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}

	// Get current image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeIngredientImageURL(ctx, request.IngredientID)
	if err != nil {
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Deleting old image
	if oldImage.Valid {
		env.Logger.DebugContext(ctx, "deleting current image")
		err = env.FileStore.DeleteURLPath(oldImage.String)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to delete old image")
			return PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
				Status:  apiError.InternalServerError.StatusCode(),
				Code:    apiError.InternalServerError.String(),
				Message: "Internal Server Error",
				ErrorId: requestID,
			}, nil
		}
	}

	// Write new image
	env.Logger.DebugContext(ctx, "writing new image")
	urlpath, _, err := env.FileStore.WriteIngredientImage(request.RecipeID,
		request.IngredientID, file.Suffix, file.Data)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to write ingredient image", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Update image url in database
	env.Logger.DebugContext(ctx, "update image in database")
	ingredient, err := env.Database.UpdateRecipeIngredient(ctx, database.UpdateRecipeIngredientParams{
		ID: request.IngredientID,
		ImageUrl: pgtype.Text{
			String: urlpath,
			Valid:  true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe ingredient", slog.Any("error", err))
		return PostApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	imageURL := env.FileStore.FileURL(ingredient.ImageUrl.String)
	res := PostApiRecipesRecipeIDIngredientsIngredientIDImage200JSONResponse{
		Id:       ingredient.ID,
		ImageUrl: &imageURL,
		Name:     ingredient.Name.String,
	}
	if ingredient.Unit.Valid {
		unit := ingredient.Unit.String
		res.Unit = &unit
	}
	if ingredient.Quantity.Valid {
		quantity := ingredient.Quantity.Float32
		res.Quantity = &quantity
	}
	return res, nil
}

func (Server) DeleteApiRecipesRecipeIDIngredientsIngredientIDImage(ctx context.Context,
	request DeleteApiRecipesRecipeIDIngredientsIngredientIDImageRequestObject) (
	DeleteApiRecipesRecipeIDIngredientsIngredientIDImageResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsIngredient, err := env.Database.CheckIngredientOwnership(ctx, database.CheckIngredientOwnershipParams{
		RecipeID:     request.RecipeID,
		IngredientID: request.IngredientID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsIngredient {
		env.Logger.ErrorContext(ctx, "user does not own recipe or ingredient")
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/ingredient does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Get Image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeIngredientImageURL(ctx, request.IngredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get current image url", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	if !oldImage.Valid {
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage404JSONResponse{
			Status:  apiError.ImageNotFound.StatusCode(),
			Code:    apiError.ImageNotFound.String(),
			Message: "image not found",
			ErrorId: requestID,
		}, nil
	}

	// Delete from database
	env.Logger.DebugContext(ctx, "deleting image from database")
	err = env.Database.DeleteRecipeIngredientImageURL(ctx, request.IngredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete image from database", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Delete file from system
	env.Logger.DebugContext(ctx, "deleting current image")
	err = env.FileStore.DeleteURLPath(oldImage.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete old image", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage204Response{}, nil
}

func (Server) PostApiRecipesRecipeIDSteps(ctx context.Context,
	request PostApiRecipesRecipeIDStepsRequestObject,
) (PostApiRecipesRecipeIDStepsResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipesRecipeIDSteps400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsStep, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step ownership", slog.Any("error", err))
		return PostApiRecipesRecipeIDSteps500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsStep {
		env.Logger.ErrorContext(ctx, "user does not own recipe")
		return PostApiRecipesRecipeIDSteps404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Create step
	env.Logger.DebugContext(ctx, "creating step")
	step, err := env.Database.CreateRecipeStep(ctx, database.CreateRecipeStepParams{
		RecipeID: request.RecipeID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create step", slog.Any("error", err))
		return PostApiRecipesRecipeIDSteps500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiRecipesRecipeIDSteps200JSONResponse{
		Id:         step.ID,
		StepNumber: step.StepNumber,
	}, nil
}

func (Server) PatchApiRecipesRecipeIDStepsStepID(ctx context.Context,
	request PatchApiRecipesRecipeIDStepsStepIDRequestObject,
) (PatchApiRecipesRecipeIDStepsStepIDResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PatchApiRecipesRecipeIDStepsStepID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsStep, err := env.Database.CheckStepOwnership(ctx, database.CheckStepOwnershipParams{
		RecipeID: request.RecipeID,
		StepID:   request.StepID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step ownership", slog.Any("error", err))
		return PatchApiRecipesRecipeIDStepsStepID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsStep {
		env.Logger.ErrorContext(ctx, "user does not own recipe or step")
		return PatchApiRecipesRecipeIDStepsStepID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/step does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Update step
	env.Logger.DebugContext(ctx, "updating step")
	updateParams := database.UpdateRecipeStepParams{
		ID: request.StepID,
	}
	if request.Body.Instruction != nil {
		updateParams.Instruction = pgtype.Text{
			String: *request.Body.Instruction,
			Valid:  true,
		}
	}
	if request.Body.StepNumber != nil {
		updateParams.StepNumber = pgtype.Int4{
			Int32: *request.Body.StepNumber,
			Valid: true,
		}
	}
	step, err := env.Database.UpdateRecipeStep(ctx, updateParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe step", slog.Any("error", err))
		return PatchApiRecipesRecipeIDStepsStepID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Return response
	res := PatchApiRecipesRecipeIDStepsStepID200JSONResponse{
		Id:         step.ID,
		StepNumber: step.StepNumber,
	}
	if step.Instruction.Valid {
		inst := step.Instruction.String
		res.Instruction = &inst
	}
	if step.ImageUrl.Valid {
		url := env.FileStore.FileURL(step.ImageUrl.String)
		res.ImageUrl = &url
	}
	return res, nil
}

func (Server) PostApiRecipesRecipeIDStepsStepIDImage(ctx context.Context,
	request PostApiRecipesRecipeIDStepsStepIDImageRequestObject,
) (PostApiRecipesRecipeIDStepsStepIDImageResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsStep, err := env.Database.CheckStepOwnership(ctx, database.CheckStepOwnershipParams{
		RecipeID: request.RecipeID,
		StepID:   request.StepID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step ownership", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsStep {
		env.Logger.ErrorContext(ctx, "user does not own recipe or step")
		return PostApiRecipesRecipeIDStepsStepIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/step does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Read image
	env.Logger.DebugContext(ctx, "reading recipe image")
	requestForm, err := request.Body.ReadForm(form.MaximumUploadSize)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read form", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid form",
			ErrorId: requestID,
		}, nil
	}
	imageFile, err := requestForm.File["image"][0].Open()
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to open image", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}
	defer func() { _ = imageFile.Close() }()
	file, err := form.ReadFile(imageFile)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read image", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}

	// Get current image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeStepImageURL(ctx, request.StepID)
	if err != nil {
		return PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Deleting old image
	if oldImage.Valid {
		env.Logger.DebugContext(ctx, "deleting current image")
		err = env.FileStore.DeleteURLPath(oldImage.String)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to delete old image")
			return PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
				Status:  apiError.InternalServerError.StatusCode(),
				Code:    apiError.InternalServerError.String(),
				Message: "Internal Server Error",
				ErrorId: requestID,
			}, nil
		}
	}

	// Write new image
	env.Logger.DebugContext(ctx, "writing new image")
	urlpath, _, err := env.FileStore.WriteStepImage(request.RecipeID,
		request.StepID, file.Suffix, file.Data)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to write step image", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Update image url in database
	env.Logger.DebugContext(ctx, "update image in database")
	step, err := env.Database.UpdateRecipeStep(ctx, database.UpdateRecipeStepParams{
		ID: request.StepID,
		ImageUrl: pgtype.Text{
			String: urlpath,
			Valid:  true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe step", slog.Any("error", err))
		return PostApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	imageURL := env.FileStore.FileURL(step.ImageUrl.String)
	res := PostApiRecipesRecipeIDStepsStepIDImage200JSONResponse{
		Id:         step.ID,
		StepNumber: step.StepNumber,
		ImageUrl:   &imageURL,
	}
	if step.Instruction.Valid {
		inst := step.Instruction.String
		res.Instruction = &inst
	}
	return res, nil
}

func (Server) DeleteApiRecipesRecipeIDStepsStepIDImage(ctx context.Context,
	request DeleteApiRecipesRecipeIDStepsStepIDImageRequestObject) (
	DeleteApiRecipesRecipeIDStepsStepIDImageResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsStep, err := env.Database.CheckStepOwnership(ctx, database.CheckStepOwnershipParams{
		RecipeID: request.RecipeID,
		StepID:   request.StepID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsStep {
		env.Logger.ErrorContext(ctx, "user does not own recipe or step")
		return DeleteApiRecipesRecipeIDStepsStepIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/step does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Get Image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeStepImageURL(ctx, request.StepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get image url", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	if !oldImage.Valid {
		return DeleteApiRecipesRecipeIDStepsStepIDImage404JSONResponse{
			Status:  apiError.ImageNotFound.StatusCode(),
			Code:    apiError.ImageNotFound.String(),
			Message: "image not found",
			ErrorId: requestID,
		}, nil
	}

	// Delete from database
	env.Logger.DebugContext(ctx, "deleting image from database")
	err = env.Database.DeleteRecipeStepImageURL(ctx, request.StepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete image from database", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Delete file from system
	env.Logger.DebugContext(ctx, "deleting current image")
	err = env.FileStore.DeleteURLPath(oldImage.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete old image", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDStepsStepIDImage204Response{}, nil
}

func (Server) DeleteApiRecipesRecipeIDIngredientsIngredientID(ctx context.Context,
	request DeleteApiRecipesRecipeIDIngredientsIngredientIDRequestObject) (
	DeleteApiRecipesRecipeIDIngredientsIngredientIDResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsIngredient, err := env.Database.CheckIngredientOwnership(ctx, database.CheckIngredientOwnershipParams{
		RecipeID:     request.RecipeID,
		IngredientID: request.IngredientID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check recipe ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsIngredient {
		env.Logger.ErrorContext(ctx, "user does not own recipe or ingredient")
		return DeleteApiRecipesRecipeIDIngredientsIngredientID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/ingredient does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Get Image URL
	env.Logger.DebugContext(ctx, "getting image url")
	imageurl, err := env.Database.GetRecipeIngredientImageURL(ctx, request.IngredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get image url", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	if imageurl.Valid {
		env.Logger.DebugContext(ctx, "deleting image")
		err = env.FileStore.DeleteURLPath(imageurl.String)
		if errors.Is(err, fileserver.ErrNotExist) {
			env.Logger.WarnContext(ctx, "image not found", slog.Any("error", err))
		} else if err != nil {
			env.Logger.ErrorContext(ctx, "failed to delete image")
			return DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
				Status:  apiError.InternalServerError.StatusCode(),
				Code:    apiError.InternalServerError.String(),
				Message: "Internal Server Error",
				ErrorId: requestID,
			}, nil
		}
	}

	// Delete ingredient from database
	env.Logger.DebugContext(ctx, "deleting ingredient")
	err = env.Database.DeleteRecipeIngredient(ctx, request.IngredientID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete ingredient", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDIngredientsIngredientID204Response{}, nil
}

func (Server) GetApiRecipesPublic(ctx context.Context,
	request GetApiRecipesPublicRequestObject) (
	GetApiRecipesPublicResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// TODO: add pagination
	env.Logger.DebugContext(ctx, "getting public recipes")
	rows, err := env.Database.GetPublicRecipes(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get public recipes", slog.Any("error", err))
		return GetApiRecipesPublic500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Build response
	res := GetApiRecipesPublic200JSONResponse{
		Recipes: make([]RecipeAndOwner, len(rows)),
	}
	for idx, recipe := range rows {
		r := Recipe{
			CreatedAt: recipe.CreatedAt.Time,
			UpdatedAt: recipe.UpdatedAt.Time,
			UserId:    recipe.UserID.Int64,
			Title:     recipe.Title,
			Published: recipe.Published,
			Id:        recipe.RecipeID,
		}
		if recipe.CookTimeAmount.Valid {
			r.CookTimeAmount = &recipe.CookTimeAmount.Int32
		}
		if recipe.CookTimeUnit.Valid {
			r.CookTimeUnit = (*TimeUnit)(&recipe.CookTimeUnit.TimeUnit)
		}
		if recipe.Description.Valid {
			r.Description = &recipe.Description.String
		}
		if recipe.ImageUrl.Valid {
			url := env.FileStore.FileURL(recipe.ImageUrl.String)
			r.ImageUrl = &url
		}
		if recipe.PrepTimeAmount.Valid {
			r.PrepTimeAmount = &recipe.PrepTimeAmount.Int32
		}
		if recipe.PrepTimeUnit.Valid {
			r.PrepTimeUnit = (*TimeUnit)(&recipe.PrepTimeUnit.TimeUnit)
		}
		if recipe.Servings.Valid {
			r.Servings = &recipe.Servings.Float32
		}

		ro := RecipeOwner{
			FirstName: recipe.FirstName,
			LastName:  recipe.LastName,
			Id:        recipe.UserID.Int64,
		}

		res.Recipes[idx] = RecipeAndOwner{
			Recipe: &r,
			Owner:  &ro,
		}
	}

	return res, nil
}

func (Server) DeleteApiRecipesRecipeIDStepsStepID(ctx context.Context,
	request DeleteApiRecipesRecipeIDStepsStepIDRequestObject) (
	DeleteApiRecipesRecipeIDStepsStepIDResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking user ownership")
	ownsStep, err := env.Database.CheckStepOwnership(ctx, database.CheckStepOwnershipParams{
		RecipeID: request.RecipeID,
		StepID:   request.StepID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check step ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsStep {
		env.Logger.ErrorContext(ctx, "user does not own recipe or step")
		return DeleteApiRecipesRecipeIDStepsStepID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe/step does not exist or user does not own recipe",
			ErrorId: requestID,
		}, nil
	}

	// Get imageurl
	env.Logger.DebugContext(ctx, "getting image url")
	imageurl, err := env.Database.GetRecipeStepImageURL(ctx, request.StepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get image url")
		return DeleteApiRecipesRecipeIDStepsStepID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	if imageurl.Valid {
		env.Logger.DebugContext(ctx, "deleting image url")
		err = env.FileStore.DeleteURLPath(imageurl.String)
		if errors.Is(err, fileserver.ErrNotExist) {
			env.Logger.WarnContext(ctx, "image not found", slog.Any("error", err))
		} else if err != nil {
			env.Logger.ErrorContext(ctx, "failed to delete image")
			return DeleteApiRecipesRecipeIDStepsStepID500JSONResponse{
				Status:  apiError.InternalServerError.StatusCode(),
				Code:    apiError.InternalServerError.String(),
				Message: "Internal Server Error",
				ErrorId: requestID,
			}, nil
		}
	}

	// Delete step
	env.Logger.DebugContext(ctx, "deleting recipe step")
	err = env.Database.DeleteRecipeStep(ctx, request.StepID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete step", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDStepsStepID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDStepsStepID204Response{}, nil
}

func (Server) GetApiRecipes(ctx context.Context,
	request GetApiRecipesRequestObject) (
	GetApiRecipesResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return GetApiRecipes400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Get user recipes
	env.Logger.DebugContext(ctx, "getting user recipes")
	rows, err := env.Database.GetRecipesByOwner(ctx, userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user recipes", slog.Any("error", err))
		return GetApiRecipes500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Build response
	env.Logger.DebugContext(ctx, "building response")
	res := GetApiRecipes200JSONResponse{
		Recipes: make([]RecipeAndOwner, len(rows)),
	}
	for idx, recipe := range rows {
		r := Recipe{
			CreatedAt: recipe.CreatedAt.Time,
			UpdatedAt: recipe.UpdatedAt.Time,
			UserId:    recipe.UserID.Int64,
			Title:     recipe.Title,
			Published: recipe.Published,
			Id:        recipe.RecipeID,
		}
		if recipe.CookTimeAmount.Valid {
			r.CookTimeAmount = &recipe.CookTimeAmount.Int32
		}
		if recipe.CookTimeUnit.Valid {
			r.CookTimeUnit = (*TimeUnit)(&recipe.CookTimeUnit.TimeUnit)
		}
		if recipe.Description.Valid {
			r.Description = &recipe.Description.String
		}
		if recipe.ImageUrl.Valid {
			url := env.FileStore.FileURL(recipe.ImageUrl.String)
			r.ImageUrl = &url
		}
		if recipe.PrepTimeAmount.Valid {
			r.PrepTimeAmount = &recipe.PrepTimeAmount.Int32
		}
		if recipe.PrepTimeUnit.Valid {
			r.PrepTimeUnit = (*TimeUnit)(&recipe.PrepTimeUnit.TimeUnit)
		}
		if recipe.Servings.Valid {
			r.Servings = &recipe.Servings.Float32
		}

		ro := RecipeOwner{
			FirstName: recipe.FirstName,
			LastName:  recipe.LastName,
			Id:        recipe.UserID.Int64,
		}

		res.Recipes[idx] = RecipeAndOwner{
			Recipe: &r,
			Owner:  &ro,
		}
	}

	return res, nil
}

func (Server) PatchApiRecipesRecipeID(ctx context.Context,
	request PatchApiRecipesRecipeIDRequestObject) (
	PatchApiRecipesRecipeIDResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PatchApiRecipesRecipeID400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// check ownership
	env.Logger.DebugContext(ctx, "checking recipe ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check ownership", slog.Any("error", err))
		return PatchApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		return PatchApiRecipesRecipeID404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe not owned by user or does not exist",
			ErrorId: requestID,
		}, nil
	}

	// Update recipe
	env.Logger.DebugContext(ctx, "updating recipe")
	updateParams := database.UpdateRecipeParams{
		ID: request.RecipeID,
	}
	if request.Body.Description != nil {
		updateParams.Description.String = *request.Body.Description
		updateParams.Description.Valid = true
	}
	if request.Body.Servings != nil {
		updateParams.Servings.Float32 = *request.Body.Servings
		updateParams.Servings.Valid = true
	}
	if request.Body.PrepTimeAmount != nil {
		updateParams.PrepTimeAmount.Int32 = *request.Body.PrepTimeAmount
		updateParams.PrepTimeAmount.Valid = true
	}
	if request.Body.PrepTimeUnit != nil {
		updateParams.PrepTimeUnit.TimeUnit = database.TimeUnit(*request.Body.PrepTimeUnit)
		updateParams.PrepTimeUnit.Valid = true
	}
	if request.Body.CookTimeAmount != nil {
		updateParams.CookTimeAmount.Int32 = *request.Body.CookTimeAmount
		updateParams.CookTimeAmount.Valid = true
	}
	if request.Body.CookTimeUnit != nil {
		updateParams.CookTimeUnit.TimeUnit = database.TimeUnit(*request.Body.CookTimeUnit)
		updateParams.CookTimeUnit.Valid = true
	}
	if request.Body.Published != nil {
		updateParams.Published.Bool = *request.Body.Published
		updateParams.Published.Valid = true
	}
	if request.Body.Title != nil {
		updateParams.Title.String = *request.Body.Title
		updateParams.Title.Valid = true
	}
	rec, err := env.Database.UpdateRecipe(ctx, updateParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe", slog.Any("error", err))
		return PatchApiRecipesRecipeID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	resp := PatchApiRecipesRecipeID200JSONResponse{
		Id:        rec.ID,
		Published: rec.Published,
		Title:     rec.Title,
		UserId:    userID,
		CreatedAt: rec.CreatedAt.Time,
		UpdatedAt: rec.UpdatedAt.Time,
	}
	if rec.CookTimeUnit.Valid {
		unit := TimeUnit(rec.CookTimeUnit.TimeUnit)
		resp.CookTimeUnit = &unit
	}
	if rec.CookTimeAmount.Valid {
		am := rec.CookTimeAmount.Int32
		resp.CookTimeAmount = &am
	}
	if rec.PrepTimeUnit.Valid {
		unit := TimeUnit(rec.PrepTimeUnit.TimeUnit)
		resp.PrepTimeUnit = &unit
	}
	if rec.PrepTimeAmount.Valid {
		am := rec.PrepTimeAmount.Int32
		resp.PrepTimeAmount = &am
	}
	if rec.Servings.Valid {
		servings := rec.Servings.Float32
		resp.Servings = &servings
	}
	if rec.Description.Valid {
		desc := rec.Description.String
		resp.Description = &desc
	}
	if rec.ImageUrl.Valid {
		imageURL := env.FileStore.FileURL(rec.ImageUrl.String)
		resp.ImageUrl = &imageURL
	}

	return resp, nil
}

func (Server) PostApiRecipesRecipeIDImage(ctx context.Context,
	request PostApiRecipesRecipeIDImageRequestObject,
) (PostApiRecipesRecipeIDImageResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// check ownership
	env.Logger.DebugContext(ctx, "checking recipe ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check ownership", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		return PostApiRecipesRecipeIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe not owned by user or does not exist",
			ErrorId: requestID,
		}, nil
	}

	// Read image
	env.Logger.DebugContext(ctx, "reading recipe image")
	requestForm, err := request.Body.ReadForm(form.MaximumUploadSize)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read form", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid form",
			ErrorId: requestID,
		}, nil
	}
	imageFile, err := requestForm.File["image"][0].Open()
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to open image", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}
	defer func() { _ = imageFile.Close() }()
	file, err := form.ReadFile(imageFile)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to read image", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "invalid image",
			ErrorId: requestID,
		}, nil
	}

	// Get current image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeImageURL(ctx, request.RecipeID)
	if err != nil {
		return PostApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Deleting old image
	if oldImage.Valid {
		env.Logger.DebugContext(ctx, "deleting current image")
		err = env.FileStore.DeleteURLPath(oldImage.String)
		if err != nil {
			env.Logger.ErrorContext(ctx, "failed to delete old image")
			return PostApiRecipesRecipeIDImage500JSONResponse{
				Status:  apiError.InternalServerError.StatusCode(),
				Code:    apiError.InternalServerError.String(),
				Message: "Internal Server Error",
				ErrorId: requestID,
			}, nil
		}
	}

	// Write new image
	env.Logger.DebugContext(ctx, "writing new image")
	urlpath, _, err := env.FileStore.WriteRecipeCoverImage(request.RecipeID,
		file.Suffix, file.Data)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to write ingredient image", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Update image url in database
	env.Logger.DebugContext(ctx, "update image in database")
	rec, err := env.Database.UpdateRecipe(ctx, database.UpdateRecipeParams{
		ID: request.RecipeID,
		ImageUrl: pgtype.Text{
			String: urlpath,
			Valid:  true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update recipe", slog.Any("error", err))
		return PostApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	resp := PostApiRecipesRecipeIDImage200JSONResponse{
		Id:        rec.ID,
		Published: rec.Published,
		Title:     rec.Title,
		UserId:    userID,
		CreatedAt: rec.CreatedAt.Time,
		UpdatedAt: rec.UpdatedAt.Time,
	}
	if rec.CookTimeUnit.Valid {
		unit := TimeUnit(rec.CookTimeUnit.TimeUnit)
		resp.CookTimeUnit = &unit
	}
	if rec.CookTimeAmount.Valid {
		am := rec.CookTimeAmount.Int32
		resp.CookTimeAmount = &am
	}
	if rec.PrepTimeUnit.Valid {
		unit := TimeUnit(rec.PrepTimeUnit.TimeUnit)
		resp.PrepTimeUnit = &unit
	}
	if rec.PrepTimeAmount.Valid {
		am := rec.PrepTimeAmount.Int32
		resp.PrepTimeAmount = &am
	}
	if rec.Servings.Valid {
		servings := rec.Servings.Float32
		resp.Servings = &servings
	}
	if rec.Description.Valid {
		desc := rec.Description.String
		resp.Description = &desc
	}
	if rec.ImageUrl.Valid {
		imageURL := env.FileStore.FileURL(rec.ImageUrl.String)
		resp.ImageUrl = &imageURL
	}

	return resp, nil
}

func (Server) DeleteApiRecipesRecipeIDImage(ctx context.Context,
	request DeleteApiRecipesRecipeIDImageRequestObject,
) (DeleteApiRecipesRecipeIDImageResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDImage404JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing user id",
			ErrorId: requestID,
		}, nil
	}

	// Check ownership
	env.Logger.DebugContext(ctx, "checking recipe ownership")
	ownsRecipe, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
		ID: request.RecipeID,
		UserID: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to check ownership", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
		return DeleteApiRecipesRecipeIDImage404JSONResponse{
			Status:  apiError.RecipeNotFound.StatusCode(),
			Code:    apiError.RecipeNotFound.String(),
			Message: "recipe not owned by user or does not exist",
			ErrorId: requestID,
		}, nil
	}

	// Get current image
	env.Logger.DebugContext(ctx, "getting current image url")
	oldImage, err := env.Database.GetRecipeImageURL(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get current image url", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	if !oldImage.Valid {
		return DeleteApiRecipesRecipeIDImage404JSONResponse{
			Status:  apiError.ImageNotFound.StatusCode(),
			Code:    apiError.ImageNotFound.String(),
			Message: "image not found",
			ErrorId: requestID,
		}, nil
	}

	// Delete from database
	env.Logger.DebugContext(ctx, "deleting image from database")
	err = env.Database.UpdateRecipeCoverImage(ctx, database.UpdateRecipeCoverImageParams{
		ID:       request.RecipeID,
		ImageUrl: pgtype.Text{Valid: false},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete image from database", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Delete file from system
	env.Logger.DebugContext(ctx, "deleting current image")
	err = env.FileStore.DeleteURLPath(oldImage.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete old image", slog.Any("error", err))
		return DeleteApiRecipesRecipeIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDImage204Response{}, nil
}
