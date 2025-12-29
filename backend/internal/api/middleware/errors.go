package middleware

import "errors"

var (
	ErrMissingCSRFHeader = errors.New("missing csrf header")
	ErrMissingCSRFCookie = errors.New("missing csrf cookie")
	ErrCSRFTokenMismatch = errors.New("csrf token mismatch")
)
