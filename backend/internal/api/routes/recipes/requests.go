package recipes

import (
	"errors"
	"strconv"
)

type (
	integer64 string
	integer32 string
	quantity  string
	timeUnit  string
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

func (t timeUnit) Validate() error {
	switch string(t) {
	case "minutes", "hours", "days":
		return nil
	default:
		return errors.New("time unit must be one of: minutes, hours, days")
	}
}

func (t timeUnit) String() string {
	return string(t)
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
	Title          string    `validate:"omitempty"`
	Description    string    `validate:"omitempty"`
	Published      string    `validate:"omitempty,boolean"`
	CookTimeAmount integer32 `validate:"omitempty,validateFn"`
	CookTimeUnit   string    `validate:"omitempty,oneof=minutes hours days"`
	PrepTimeAmount integer32 `validate:"omitempty,validateFn"`
	PrepTimeUnit   string    `validate:"omitempty,oneof=minutes hours days"`
	Servings       string    `validate:"omitempty,numeric"`
}

type GetRecipeRequest struct {
	RecipeID integer64 `validate:"required"`
}

type UpdateRecipeFullData struct {
	Title          *string                          `json:"title" validate:"omitempty"`
	Description    *string                          `json:"description" validate:"omitempty"`
	Published      *bool                            `json:"published" validate:"omitempty"`
	CookTimeAmount *int32                           `json:"cook_time_amount" validate:"omitempty,gte=0"`
	CookTimeUnit   *timeUnit                        `json:"cook_time_unit" validate:"omitempty,validateFn"`
	PrepTimeAmount *int32                           `json:"prep_time_amount" validate:"omitempty,gte=0"`
	PrepTimeUnit   *timeUnit                        `json:"prep_time_unit" validate:"omitempty,validateFn"`
	Servings       *float32                         `json:"servings" validate:"omitempty,gt=0"`
	Ingredients    []UpdateRecipeFullDataIngredient `json:"ingredients" validate:"omitempty,dive"`
	Steps          []UpdateRecipeFullDataStep       `json:"steps" validate:"omitempty,dive"`
}

type UpdateRecipeFullDataIngredient struct {
	ID       *int64  `json:"id" validate:"omitempty"`
	Quantity float32 `json:"quantity" validate:"required,gt=0"`
	Unit     *string `json:"unit" validate:"omitempty"`
	Name     string  `json:"name" validate:"required"`
}

type UpdateRecipeFullDataStep struct {
	ID          *int64 `json:"id" validate:"omitempty"`
	Instruction string `json:"instruction" validate:"required"`
}
