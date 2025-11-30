package recipes

import (
	"errors"
	"strconv"
)

type (
	recipeID     string
	ingredientID string
	stepID       string
	quantity     string
)

func (r recipeID) Validate() error {
	v, err := strconv.ParseInt(string(r), 10, 64)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("recipe id should be non-negative")
	}
	return nil
}

func (i ingredientID) Validate() error {
	v, err := strconv.ParseInt(string(i), 10, 64)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("ingredient id should be non-negative")
	}
	return nil
}

func (i stepID) Validate() error {
	v, err := strconv.ParseInt(string(i), 10, 64)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("step id should be non-negative")
	}
	return nil
}

func (q quantity) Validate() error {
	v, err := strconv.ParseFloat(string(q), 32)
	if err != nil {
		return errors.New("expected a float")
	}
	if v <= 0.0 {
		return errors.New("quantity should be non-negative")
	}
	return nil
}

type CreateIngredientRequest struct {
	RecipeID recipeID `validate:"required,numeric"`
	Name     string   `validate:"required"`
	Quantity quantity `validate:"required,numeric"`
	Unit     string   `validate:"omitempty"`
}

type CreateRecipeStepRequest struct {
	RecipeID    recipeID `validate:"required,numeric"`
	Instruction string   `validate:"required"`
}
