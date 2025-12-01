package error

import "net/http"

type ErrorCode string

const (
	InternalServerError     = "internal_server_error"
	BadRequest              = "bad_request"
	UnprocessibleEntity     = "unprocessible_entity"
	InvalidAccessToken      = "invalid_access_token"
	ExpiredAccessToken      = "expired_access_token"
	InvalidRefreshToken     = "invalid_refresh_token"
	ExpiredRefreshToken     = "expired_refresh_token"
	InsufficientPermissions = "insufficient_permissions"
	WeakPassword            = "weak_password"
	EmailConflict           = "email_conflict"
	AdminAlreadySetup       = "admin_already_setup"
	RecipeNotFound          = "recipe_not_found"
	RecipeNotOwned          = "recipe_not_owned"
	IngredientNotFound      = "ingredient_not_found"
	StepNotFound            = "step_not_found"
)

var errorCodeToStatusCode = map[ErrorCode]int{
	InternalServerError:     http.StatusInternalServerError,
	BadRequest:              http.StatusBadRequest,
	UnprocessibleEntity:     http.StatusUnprocessableEntity,
	InvalidAccessToken:      http.StatusUnauthorized,
	ExpiredAccessToken:      http.StatusUnauthorized,
	InsufficientPermissions: http.StatusForbidden,
	WeakPassword:            http.StatusUnprocessableEntity,
	EmailConflict:           http.StatusConflict,
	AdminAlreadySetup:       http.StatusConflict,
	RecipeNotFound:          http.StatusNotFound,
	RecipeNotOwned:          http.StatusForbidden,
	IngredientNotFound:      http.StatusNotFound,
	StepNotFound:            http.StatusNotFound,
	InvalidRefreshToken:     http.StatusUnauthorized,
	ExpiredRefreshToken:     http.StatusUnauthorized,
}
