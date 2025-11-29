// Package token contains utilities for http tokens.
package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/jwt"
)

const (
	accessTokenBytes     = 32
	accessTokenLifetime  = 60 * 30           // 30 minutes
	refreshTokenLifetime = 60 * 60 * 24 * 14 // 14 days
)

func AccessTokenName(env *env.Env) string {
	if env.Get("ENV") == "production" {
		return "__Host-Http-access"
	}
	return "access"
}

func RefreshTokenName(env *env.Env) string {
	if env.Get("ENV") == "production" {
		return "__Host-Http-refresh"
	}
	return "refresh"
}

func CreateToken(numbytes uint) (string, error) {
	token := make([]byte, numbytes)
	if _, err := rand.Reader.Read(token); err != nil {
		return "", fmt.Errorf("creating token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(token), nil
}

func CreateRefreshToken(userid string) (string, error) {
	randSegment, err := CreateToken(accessTokenBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", userid, randSegment), nil
}

func CreateAccessToken(params jwt.JWTParams, env *env.Env) (string, error) {
	secret := env.Get("APP_SECRET")
	if secret == "" {
		return "", errors.New("environment variable APP_SECRET not defined")
	}
	token, err := jwt.GenerateJWT(params, []byte(secret), "1")
	if err != nil {
		return "", fmt.Errorf("generating access token: %w", err)
	}
	return token, nil
}

func NewAccessTokenCookie(token string, env *env.Env) *http.Cookie {
	cookie := &http.Cookie{
		Name:     AccessTokenName(env),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   accessTokenLifetime,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	}

	if env.Get("ENV") == "production" {
		cookie.Secure = true
	}

	return cookie
}

func NewRefreshTokenCookie(token string, env *env.Env) *http.Cookie {
	cookie := &http.Cookie{
		Name:     RefreshTokenName(env),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   refreshTokenLifetime,
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	}

	if os.Getenv("ENV") == "production" {
		cookie.Secure = true
	}

	return cookie
}
