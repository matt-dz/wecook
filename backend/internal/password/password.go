// Package password contains utilities for managing passwords.
package password

import (
	"errors"
	"regexp"

	passwordvalidator "github.com/wagslane/go-password-validator"
)

const (
	minimumLength       = 10
	minimumEntropoyBits = 60
)

var (
	uppercaseRe = regexp.MustCompile(`[A-Z]`)
	lowercaseRe = regexp.MustCompile(`[a-z]`)
	digitRe     = regexp.MustCompile(`[0-9]`)
	specialRe   = regexp.MustCompile(`[!@#$%^&*()\-_=+{};:,.<>/?\\|"']`)
)

var (
	ErrTooShort    = errors.New("password must be at least 10 characters long")
	ErrNoUppercase = errors.New("password must contain at least one uppercase letter")
	ErrNoLowercase = errors.New("password must contain at least one lowercase letter")
	ErrNoDigit     = errors.New("password must contain at least one digit")
	ErrNoSpecial   = errors.New("password must contain at least one special character")
	ErrTooWeak     = errors.New("password is too weak")
)

func ValidatePassword(password string) error {
	if len(password) < minimumLength {
		return ErrTooShort
	}

	if !uppercaseRe.MatchString(password) {
		return ErrNoUppercase
	}
	if !lowercaseRe.MatchString(password) {
		return ErrNoLowercase
	}
	if !digitRe.MatchString(password) {
		return ErrNoDigit
	}
	if !specialRe.MatchString(password) {
		return ErrNoSpecial
	}

	if err := passwordvalidator.Validate(password, minimumEntropoyBits); err != nil {
		return errors.Join(ErrTooWeak, err)
	}

	return nil
}
