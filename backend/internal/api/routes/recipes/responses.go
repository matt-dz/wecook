package recipes

import (
	"github.com/matt-dz/wecook/internal/recipe"
)

type CreateRecipeResponse struct {
	RecipeID int64 `json:"recipe_id"`
}

type GetRecipeResponse recipe.RecipeWithIngredientsAndStepsAndOwner

type GetPersonalRecipesResponse struct {
	Recipes []recipe.RecipeAndOwner `json:"recipes"`
}
