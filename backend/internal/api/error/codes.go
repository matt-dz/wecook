package error

import "net/http"

type ErrorCode string

const (
	UnknownError            ErrorCode = "unknown_error"
	InternalServerError     ErrorCode = "internal_server_error"
	BadRequest              ErrorCode = "bad_request"
	UnprocessibleEntity     ErrorCode = "unprocessible_entity"
	InvalidCredentials      ErrorCode = "invalid_credentials"
	InvalidAccessToken      ErrorCode = "invalid_access_token"
	ExpiredAccessToken      ErrorCode = "expired_access_token"
	InvalidRefreshToken     ErrorCode = "invalid_refresh_token"
	ExpiredRefreshToken     ErrorCode = "expired_refresh_token"
	InsufficientPermissions ErrorCode = "insufficient_permissions"
	WeakPassword            ErrorCode = "weak_password"
	EmailConflict           ErrorCode = "email_conflict"
	AdminAlreadySetup       ErrorCode = "admin_already_setup"
	RecipeNotFound          ErrorCode = "recipe_not_found"
	RecipeNotOwned          ErrorCode = "recipe_not_owned"
	IngredientNotFound      ErrorCode = "ingredient_not_found"
	StepNotFound            ErrorCode = "step_not_found"
	ImageNotFound           ErrorCode = "image_not_found"
	UserNotFound            ErrorCode = "user_not_found"
	InvalidInviteCode       ErrorCode = "invalid_invite_code"
	InvalidPassword         ErrorCode = "invalid_password"
)

var errorCodeToStatusCode = map[ErrorCode]int{
	UnknownError:            0, // No error code - unknown
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
	InvalidCredentials:      http.StatusUnauthorized,
	ImageNotFound:           http.StatusNotFound,
	UserNotFound:            http.StatusNotFound,
	InvalidInviteCode:       http.StatusUnprocessableEntity,
	InvalidPassword:         http.StatusUnprocessableEntity,
}

func (ec ErrorCode) StatusCode() int {
	return errorCodeToStatusCode[ec]
}

func (ec ErrorCode) String() string {
	return string(ec)
}
