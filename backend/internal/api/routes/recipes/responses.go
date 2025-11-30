package recipes

import "time"

type CreateRecipeResponse struct {
	RecipeID int64 `json:"recipe_id"`
}

type RecipeResponseRecipe struct {
	UserID           int64     `json:"user_id"`
	CookeTimeMinutes uint      `json:"cook_time_minutes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ImageURL         string    `json:"image_url,omitempty"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
}

type RecipeResponseUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type GetRecipeResponse struct {
	Recipe RecipeResponseRecipe `json:"recipe"`
	User   RecipeResponseUser   `json:"user"`
}
