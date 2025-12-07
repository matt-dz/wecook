package client

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/role"
)

func (Server) PostApiAuthRefresh(ctx context.Context,
	request PostApiAuthRefreshRequestObject,
) (PostApiAuthRefreshResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	var refreshToken string
	if request.Body != nil && request.Body.RefreshToken != nil {
		refreshToken = *request.Body.RefreshToken
	} else if request.Params.Refresh != nil {
		refreshToken = *request.Params.Refresh
	}
	if refreshToken == "" {
		env.Logger.ErrorContext(ctx, "refresh token not provided")
		return PostApiAuthRefresh401JSONResponse{
			Status:  apiError.InvalidRefreshToken.StatusCode(),
			Code:    apiError.InvalidRefreshToken.String(),
			Message: "refresh token not provied",
			ErrorId: requestID,
		}, nil
	}

	// Extract UserID from refresh token
	env.Logger.DebugContext(ctx, "Extracting user id from refresh token")
	userID, err := token.ExtractUserIDFromRefreshToken(refreshToken)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from refresh token", slog.Any("error", err))
		return PostApiAuthRefresh401JSONResponse{
			Status:  apiError.InvalidRefreshToken.StatusCode(),
			Code:    apiError.InvalidRefreshToken.String(),
			Message: "malformed refresh token",
			ErrorId: requestID,
		}, nil
	}

	// Retrieve true refresh token
	env.Logger.DebugContext(ctx, "Fetching true refresh token")
	refresh, err := env.Database.GetUserRefreshTokenHash(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "no user with given id", slog.Int64("user-id", userID), slog.Any("error", err))
		return PostApiAuthRefresh401JSONResponse{
			Status:  apiError.InvalidRefreshToken.StatusCode(),
			Code:    apiError.InvalidRefreshToken.String(),
			Message: "invalid refresh token",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user refresh hash", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Decode true hash
	env.Logger.DebugContext(ctx, "Decoding true refresh token hash")
	argonParams, argonSalt, trueRefreshHash, err := argon2id.DecodeHash(refresh.RefreshTokenHash.String)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to decode hash", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Hash given refresh token
	env.Logger.DebugContext(ctx, "Encoding given refresh token")
	givenHash := argon2id.HashWithSalt(refreshToken, *argonParams, argonSalt)

	// Compare refresh tokens
	env.Logger.DebugContext(ctx, "Comparing tokens")
	if subtle.ConstantTimeCompare(trueRefreshHash, givenHash) == 0 {
		env.Logger.ErrorContext(ctx, "tokens do not match")
		return PostApiAuthRefresh401JSONResponse{
			Status:  apiError.InvalidRefreshToken.StatusCode(),
			Code:    apiError.InvalidRefreshToken.String(),
			Message: "invalid refresh token",
			ErrorId: requestID,
		}, nil
	}
	env.Logger.DebugContext(ctx, "tokens match!")

	if time.Now().After(refresh.RefreshTokenExpiresAt.Time) {
		env.Logger.ErrorContext(ctx, "refresh token is expired")
		return PostApiAuthRefresh401JSONResponse{
			Status:  apiError.InvalidRefreshToken.StatusCode(),
			Code:    apiError.InvalidRefreshToken.String(),
			Message: "invalid refresh token",
			ErrorId: requestID,
		}, nil
	}

	// Create new refresh token
	env.Logger.DebugContext(ctx, "creating new refresh token")
	newRefreshToken, err := token.NewRefreshToken(userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create refresh token", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Update refresh token
	env.Logger.DebugContext(ctx, "Updating refresh token")
	newRefreshTokenHash, err := argon2id.EncodeHash(newRefreshToken, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash refresh token", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
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
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Get user role
	env.Logger.DebugContext(ctx, "Getting user role")
	userRole, err := env.Database.GetUserRole(ctx, userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user role", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Generate access token
	env.Logger.DebugContext(ctx, "Generating access token")
	accessToken, err := token.NewAccessToken(jwt.JWTParams{
		Role:   role.DBToRole(userRole),
		UserID: fmt.Sprintf("%d", userID),
	}, env)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to generate access token", slog.Any("error", err))
		return PostApiAuthRefresh500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Write response
	env.Logger.DebugContext(ctx, "Writing response")
	tokenType := "Bearer"
	expiresIn := int64(token.AccessTokenLifetime)
	return loginSuccessResponse{
		accessCookie:  token.NewAccessTokenCookie(accessToken, env),
		refreshCookie: token.NewRefreshTokenCookie(refreshToken, env),
		body: LoginResponse{
			AccessToken: accessToken,
			TokenType:   &tokenType,
			ExpiresIn:   &expiresIn,
		},
	}, nil
}
