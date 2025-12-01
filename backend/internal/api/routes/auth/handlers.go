// Package auth contains handlers for the auth endpoints
package auth

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/role"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// HandleRefreshSession godoc
//
//	@Summary		Refresh session tokens
//	@Description	Uses a valid refresh token cookie to issue a new access and refresh token pair.
//	@Tags			Auth
//	@Produce		json
//
//	@Success		200	{string}	string			"New access and refresh tokens set as HTTP cookies"
//	@Failure		400	{object}	apiError.Error	"Missing or malformed refresh token cookie"
//	@Failure		401	{object}	apiError.Error	"Invalid refresh token"
//	@Failure		500	{object}	apiError.Error	"Internal server error"
//
//	@Router			/api/auth/session/refresh [post]
func HandleRefreshSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Extract refresh token
	env.Logger.DebugContext(ctx, "Extracting refresh token")
	cookie, err := r.Cookie(token.RefreshTokenName(env))
	if errors.Is(err, http.ErrNoCookie) {
		env.Logger.ErrorContext(ctx, "refresh token not found", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.MissingCredentials, "refresh token not found", requestID)
		return
	}

	// Extract UserID from refresh token
	env.Logger.DebugContext(ctx, "Extracting user id from refresh token")
	userID, err := token.ExtractUserIDFromRefreshToken(cookie.Value)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from refresh token", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "malformed refresh token", requestID)
		return
	}

	// Retrieve true refresh token
	env.Logger.DebugContext(ctx, "Fetching true refresh token")
	refresh, err := env.Database.GetUserRefreshTokenHash(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "no user with given id", slog.Int64("user-id", userID), slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "invalid refresh token", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user refresh hash", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	if time.Now().After(refresh.RefreshTokenExpiresAt.Time) {
		env.Logger.ErrorContext(ctx, "refresh token is expired")
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "invalid refresh token", requestID)
		return
	}

	// Decode true hash
	env.Logger.DebugContext(ctx, "Decoding true refresh token hash")
	argonParams, argonSalt, trueRefreshHash, err := argon2id.DecodeHash(refresh.RefreshTokenHash.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to decode hash", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Hash given refresh token
	env.Logger.DebugContext(ctx, "Encoding given refresh token")
	givenHash := argon2id.HashWithSalt(cookie.Value, *argonParams, argonSalt)

	// Compare refresh tokens
	env.Logger.DebugContext(ctx, "Comparing tokens")
	if subtle.ConstantTimeCompare(trueRefreshHash, []byte(givenHash)) != 0 {
		env.Logger.ErrorContext(ctx, "tokens do not match")
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "invalid refresh token", requestID)
		return
	}
	env.Logger.DebugContext(ctx, "tokens match!")

	// Create new refresh token
	env.Logger.DebugContext(ctx, "creating new refresh token")
	newRefreshToken, err := token.NewRefreshToken(userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create refresh token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Update refresh token
	env.Logger.DebugContext(ctx, "Updating refresh token")
	newRefreshTokenHash, err := argon2id.EncodeHash(newRefreshToken, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash refresh token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	err = env.Database.UpdateUserRefreshTokenHash(ctx, database.UpdateUserRefreshTokenHashParams{
		RefreshTokenHash: pgtype.Text{
			String: newRefreshTokenHash,
			Valid:  true,
		},
		ID: userID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update refresh token hash", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Get user role
	env.Logger.DebugContext(ctx, "Getting user role")
	userRole, err := env.Database.GetUserRole(ctx, userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user role", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Generate access token
	env.Logger.DebugContext(ctx, "Generating access token")
	accessToken, err := token.NewAccessToken(jwt.JWTParams{
		Role:   role.DBToRole(userRole),
		UserID: fmt.Sprintf("%d", userID),
	}, env)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to generate access token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.DebugContext(ctx, "Writing response")
	http.SetCookie(w, token.NewAccessTokenCookie(accessToken, env))
	http.SetCookie(w, token.NewRefreshTokenCookie(newRefreshToken, env))
}

// HandleVerifySession godoc
//
//	@Summary		Verify user session
//	@Description	Validates the user's access token cookie, checks expiration,
//	@Description	and ensures the user has the required role.
//	@Tags			Auth
//	@Accept			*/*
//	@Produce		json
//	@Success		204	"Session is valid"
//	@Failure		400	{object}	apiError.Error	"Invalid token or malformed cookie"
//	@Failure		401	{object}	apiError.Error	"Expired or invalid access token"
//	@Failure		403	{object}	apiError.Error	"Insufficient permissions"
//	@Failure		500	{object}	apiError.Error	"Internal server error"
//	@Router			/api/auth/session/verify [get]
func HandleVerifySession(w http.ResponseWriter, r *http.Request) {}
