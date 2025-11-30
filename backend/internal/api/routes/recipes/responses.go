package recipes

import "time"

type CreateRecipeResponse struct {
	RecipeID int64 `json:"recipe_id"`
}

type RecipeResponseRecipe struct {
	UserID           int64                            `json:"user_id"`
	CookeTimeMinutes uint                             `json:"cook_time_minutes"`
	CreatedAt        time.Time                        `json:"created_at"`
	UpdatedAt        time.Time                        `json:"updated_at"`
	ImageURL         string                           `json:"image_url,omitempty"`
	Title            string                           `json:"title"`
	Description      string                           `json:"description,omitempty"`
	Ingredients      []RecipeResponseRecipeIngredient `json:"ingredients"`
	Steps            []RecipeResponseRecipeStep       `json:"steps"`
}

type RecipeResponseUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RecipeResponseRecipeIngredient struct {
	ID       int64   `json:"id"`
	RecipeID int64   `json:"recipe_id"`
	Quantity float32 `json:"quantity,omitempty"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit,omitempty"`
	ImageURL string  `json:"image_url,omitempty"`
}

type RecipeResponseRecipeStep struct {
	ID          int64     `json:"id"`
	RecipeID    int64     `json:"recipe_id"`
	StepNumber  int32     `json:"step_number"`
	Instruction string    `json:"instruction"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GetRecipeResponse struct {
	Recipe RecipeResponseRecipe `json:"recipe"`
	User   RecipeResponseUser   `json:"user"`
}
