// Package token contains utilities for http tokens.
package token

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/matt-dz/wecook/internal/env"
	mJwt "github.com/matt-dz/wecook/internal/jwt"
)

type (
	accessTokenKeyType struct{}
	userIDKeyType      struct{}
)

var (
	accessTokenKey accessTokenKeyType
	userIDKey      userIDKeyType
)

const (
	AuthorizationHeader = "Authorization"
)

const (
	accessTokenBytes     = 32
	AccessTokenLifetime  = 60 * 30           // 30 minutes
	refreshTokenLifetime = 60 * 60 * 24 * 14 // 14 days
)

var ErrMalformedRefreshToken = errors.New("malformed refresh token")

func AccessTokenName() string {
	return "access"
}

func RefreshTokenName() string {
	return "refresh"
}

func CreateToken(numbytes uint) (string, error) {
	token := make([]byte, numbytes)
	if _, err := rand.Reader.Read(token); err != nil {
		return "", fmt.Errorf("creating token: %w", err)
	}
	return base64.StdEncoding.EncodeToString(token), nil
}

func NewRefreshToken(userid int64) (string, error) {
	randSegment, err := CreateToken(accessTokenBytes)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%s", userid, randSegment), nil
}

func NewAccessToken(params mJwt.JWTParams, env *env.Env) (string, error) {
	secret := env.Get("APP_SECRET")
	if secret == "" {
		return "", errors.New("environment variable APP_SECRET not defined")
	}
	token, err := mJwt.GenerateJWT(params, []byte(secret), "1")
	if err != nil {
		return "", fmt.Errorf("generating access token: %w", err)
	}
	return token, nil
}

func NewAccessTokenCookie(token string, secure bool) *http.Cookie {
	cookie := &http.Cookie{
		Name:     AccessTokenName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   AccessTokenLifetime,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func NewRefreshTokenCookie(token string, secure bool) *http.Cookie {
	cookie := &http.Cookie{
		Name:     RefreshTokenName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   refreshTokenLifetime,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}

	return cookie
}

func ExtractUserIDFromRefreshToken(token string) (int64, error) {
	userIDStr, _, found := strings.Cut(token, ".")
	if !found {
		return 0, ErrMalformedRefreshToken
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, errors.Join(ErrMalformedRefreshToken, err)
	}
	return userID, nil
}

func AccessTokenWithCtx(ctx context.Context, accessToken *jwt.Token) context.Context {
	return context.WithValue(ctx, accessTokenKey, accessToken)
}

func AccessTokenFromCtx(ctx context.Context) (*jwt.Token, error) {
	token, ok := ctx.Value(accessTokenKey).(*jwt.Token)
	if !ok {
		return nil, errors.New("invalid access token type")
	}
	return token, nil
}

func UserIDWithCtx(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromCtx(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return 0, errors.New("invalid user id type")
	}
	return userID, nil
}

func ParseBearerToken(value string) (string, error) {
	token, found := strings.CutPrefix(value, "Bearer ")
	if !found {
		return "", errors.New("token should be in format \"Bearer ...\"")
	}

	return token, nil
}
