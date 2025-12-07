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
	cookTimeUnit := TimeUnit(TimeUnit(row.CookTimeUnit.TimeUnit))
	prepTimeUnit := TimeUnit(TimeUnit(row.PrepTimeUnit.TimeUnit))
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
		imageUrl := env.FileServer.FileURL(row.ImageUrl.String)
		res.Recipe.ImageUrl = &imageUrl
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
			imageUrl := env.FileServer.FileURL(step.ImageUrl.String)
			newStep.ImageUrl = &imageUrl
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
			imageUrl := ingredient.ImageUrl.String
			newIngredient.ImageUrl = &imageUrl
		}
		res.Recipe.Ingredients = append(res.Recipe.Ingredients, newIngredient)
	}
	return res, nil
}
