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
	"github.com/matt-dz/wecook/internal/form"
)

const (
	defaultRecipeTitle = "Untitled Recipe"
	maxUploadSize      = 20 << 20 // ~ 20 MB
)

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

func (Server) GetApiRecipesRecipeID(ctx context.Context,
	request GetApiRecipesRecipeIDRequestObject,
) (GetApiRecipesRecipeIDResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Get recipe and owner
	env.Logger.DebugContext(ctx, "getting recipe and owner")
	row, err := env.Database.GetPublishedRecipeAndOwner(ctx, request.RecipeID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "recipe does not exist", slog.Any("error", err))
		return GetApiRecipesRecipeID404JSONResponse{
			Code:    apiError.RecipeNotFound.String(),
			Status:  apiError.RecipeNotFound.StatusCode(),
			Message: "recipe does not exist or is not public",
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
	steps, err := env.Database.GetRecipeSteps(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe steps", slog.Any("error", err))
		return GetApiRecipesRecipeID500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	ingredients, err := env.Database.GetRecipeIngredients(ctx, request.RecipeID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get recipe ingredients", slog.Any("error", err))
		return GetApiRecipesRecipeID500JSONResponse{
			Code:    apiError.InternalServerError.String(),
			Status:  apiError.InternalServerError.StatusCode(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Write response
	cookTimeUnit := TimeUnit(row.CookTimeUnit.TimeUnit)
	prepTimeUnit := TimeUnit(row.PrepTimeUnit.TimeUnit)
	res := GetApiRecipesRecipeID200JSONResponse{
		Owner: RecipeOwner{
			FirstName: row.FirstName,
			LastName:  row.LastName,
			Id:        row.ID,
		},
		Recipe: RecipeWithIngredientsAndSteps{
			CookTimeAmount: &row.CookTimeAmount.Int32,
			CookTimeUnit:   &cookTimeUnit,
			PrepTimeAmount: &row.PrepTimeAmount.Int32,
			PrepTimeUnit:   &prepTimeUnit,
			UserId:         row.UserID.Int64,
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
			Published:      row.Published,
			Title:          row.Title,
			Id:             row.ID,
			Servings:       &row.Servings.Float32,
			Description:    &row.Description.String,
			Steps:          make([]RecipeStep, 0),
			Ingredients:    make([]RecipeIngredient, 0),
		},
	}
	if row.ImageUrl.String != "" {
		imageURL := env.FileStore.FileURL(row.ImageUrl.String)
		res.Recipe.ImageUrl = &imageURL
	}
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
		res.Recipe.Steps = append(res.Recipe.Steps, newStep)
	}
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
		res.Recipe.Ingredients = append(res.Recipe.Ingredients, newIngredient)
	}
	return res, nil
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
		return PatchApiRecipesRecipeIDIngredientsIngredientID500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	if !ownsRecipe {
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
		res.ImageUrl = &row.ImageUrl.String
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

	return PostApiRecipesRecipeIDIngredientsIngredientIDImage200JSONResponse{
		Id:       ingredient.ID,
		ImageUrl: ingredient.ImageUrl.String,
	}, nil
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

	// Delete file from system
	env.Logger.DebugContext(ctx, "deleting current image")
	err = env.FileStore.DeleteURLPath(oldImage.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to delete old image")
		return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return DeleteApiRecipesRecipeIDIngredientsIngredientIDImage200Response{}, nil
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
	ownsIngredient, err := env.Database.CheckRecipeOwnership(ctx, database.CheckRecipeOwnershipParams{
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
	if !ownsIngredient {
		env.Logger.ErrorContext(ctx, "user does not own recipe or ingredient")
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
