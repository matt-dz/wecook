package error

import "net/http"

type ErrorCode string

const (
	InternalServerError     = "internal_server_error"
	BadRequest              = "bad_request"
	UnprocessibleEntity     = "unprocessible_entity"
	InvalidToken            = "invalid_token"
	ExpiredToken            = "expired_token"
	MissingCredentials      = "missing_credentials"
	InvalidCredentials      = "invalid_credentials"
	InsufficientPermissions = "insufficient_permissions"
)

var errorCodeToStatusCode = map[ErrorCode]int{
	InternalServerError:     http.StatusInternalServerError,
	BadRequest:              http.StatusBadRequest,
	UnprocessibleEntity:     http.StatusUnprocessableEntity,
	InvalidToken:            http.StatusUnauthorized,
	ExpiredToken:            http.StatusUnauthorized,
	MissingCredentials:      http.StatusUnauthorized,
	InvalidCredentials:      http.StatusUnauthorized,
	InsufficientPermissions: http.StatusForbidden,
}
