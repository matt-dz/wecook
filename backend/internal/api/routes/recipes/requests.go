package recipes

import (
	"errors"
	"strconv"
)

type (
	integer64 string
	integer32 string
	quantity  string
)

func (i integer64) Validate() error {
	v, err := strconv.ParseInt(i.String(), 10, 64)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("recipe id should be non-negative")
	}
	return nil
}

func (i integer64) String() string {
	return string(i)
}

func (i integer64) Int() (int64, error) {
	return strconv.ParseInt(i.String(), 10, 64)
}

func (i integer32) Validate() error {
	v, err := strconv.ParseInt(i.String(), 10, 32)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("recipe id should be non-negative")
	}
	return nil
}

func (i integer32) String() string {
	return string(i)
}

func (i integer32) Int() (int64, error) {
	return strconv.ParseInt(i.String(), 10, 32)
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
	RecipeID integer64 `validate:"required,numeric"`
	Name     string    `validate:"required"`
	Quantity quantity  `validate:"required,numeric"`
	Unit     string    `validate:"omitempty"`
}

type CreateRecipeStepRequest struct {
	RecipeID    integer64 `validate:"required,validateFn"`
	Instruction string    `validate:"required"`
}

type UpdateRecipeStepRequest struct {
	RecipeID integer64 `validate:"required,validateFn"`
	StepID   integer64 `validate:"required,validateFn"`
}

type UpdateRecipeIngredientRequest struct {
	RecipeID     integer64 `validate:"required,validateFn"`
	IngredientID integer64 `validate:"required,validateFn"`
}

type UpdateRecipeIngredientForm struct {
	Quantity string `validate:"omitempty,numeric"`
	Unit     string `validate:"omitempty"`
	Name     string `validate:"omitempty"`
}

type UpdateRecipeRequest struct {
	RecipeID integer64 `validate:"required,validateFn"`
}

type UpdateRecipeForm struct {
	Title           string    `validate:"omitempty"`
	Description     string    `validate:"omitempty"`
	Published       string    `validate:"omitempty,boolean"`
	CookTimeMinutes integer32 `validate:"omitempty,validateFn"`
	Servings        string    `validate:"omitempty,numeric"`
}

type GetRecipeRequest struct {
	RecipeID integer64 `validate:"required"`
}
