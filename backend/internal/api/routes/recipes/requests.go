package recipes

import (
	"errors"
	"strconv"
)

type (
	wecookID string
	quantity string
)

func (i wecookID) Validate() error {
	v, err := strconv.ParseInt(i.String(), 10, 64)
	if err != nil {
		return errors.New("expected an integer")
	}
	if v < 0 {
		return errors.New("recipe id should be non-negative")
	}
	return nil
}

func (i wecookID) String() string {
	return string(i)
}

func (i wecookID) Int() (int64, error) {
	return strconv.ParseInt(i.String(), 10, 64)
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
	RecipeID wecookID `validate:"required,numeric"`
	Name     string   `validate:"required"`
	Quantity quantity `validate:"required,numeric"`
	Unit     string   `validate:"omitempty"`
}

type CreateRecipeStepRequest struct {
	RecipeID    wecookID `validate:"required,validateFn"`
	Instruction string   `validate:"required"`
}

type UpdateRecipeStepRequest struct {
	RecipeID wecookID `validate:"required,validateFn"`
	StepID   wecookID `validate:"required,validateFn"`
}

type UpdateRecipeIngredientRequest struct {
	RecipeID     wecookID `validate:"required,validateFn"`
	IngredientID wecookID `validate:"required,validateFn"`
}

type UpdateRecipeIngredientForm struct {
	Quantity string `validate:"omitempty,numeric"`
	Unit     string `validate:"omitempty"`
	Name     string `validate:"omitempty"`
}
