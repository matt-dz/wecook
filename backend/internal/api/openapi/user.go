package client

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

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

// loginSuccessResponse wraps the 200 response to properly set multiple cookies and return bearer token.
type loginSuccessResponse struct {
	accessCookie  *http.Cookie
	refreshCookie *http.Cookie
	body          LoginResponse
}

func (r loginSuccessResponse) VisitPostApiLoginResponse(w http.ResponseWriter) error {
	http.SetCookie(w, r.accessCookie)
	http.SetCookie(w, r.refreshCookie)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	return encoder.Encode(r.body)
}

func (Server) PostApiLogin(ctx context.Context, request PostApiLoginRequestObject) (PostApiLoginResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Retrieve user information
	env.Logger.DebugContext(ctx, "Retrieving user information")
	user, err := env.Database.GetUser(ctx, request.Body.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx,
			"User with email does not exist",
			slog.String("email", request.Body.Email),
			slog.Any("error", err))
		return PostApiLogin401JSONResponse{
			Status:  apiError.InvalidCredentials.StatusCode(),
			Code:    apiError.InvalidCredentials.String(),
			Message: "username or password is incorrect",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to retrieve user information", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Decode user password
	env.Logger.DebugContext(ctx, "Decoding user password")
	argonParams, argonSalt, trueHash, err := argon2id.DecodeHash(user.PasswordHash)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to decode password hash", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Hash given password
	env.Logger.DebugContext(ctx, "Hashing given password")
	givenHash := argon2id.HashWithSalt(request.Body.Password, *argonParams, argonSalt)

	// Comparing passwords
	env.Logger.DebugContext(ctx, "Comparing passwords")
	if subtle.ConstantTimeCompare(givenHash, trueHash) == 0 {
		env.Logger.ErrorContext(ctx, "Given password is incorrect")
		return PostApiLogin401JSONResponse{
			Status:  apiError.InvalidCredentials.StatusCode(),
			Code:    apiError.InvalidCredentials.String(),
			Message: "username or password is incorrect",
			ErrorId: requestID,
		}, nil
	}
	env.Logger.DebugContext(ctx, "Passwords match!")

	// Create access token
	env.Logger.DebugContext(ctx, "Generating access token")
	accessToken, err := token.NewAccessToken(jwt.JWTParams{
		Role:   role.DBToRole(user.Role),
		UserID: fmt.Sprintf("%d", user.ID),
	}, env)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create access token", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Create refresh token
	env.Logger.DebugContext(ctx, "Generating refresh token")
	refreshToken, err := token.NewRefreshToken(user.ID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create refresh token", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	refreshTokenHash, err := argon2id.EncodeHash(refreshToken, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash refresh token", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	err = env.Database.UpdateUserRefreshTokenHash(ctx, database.UpdateUserRefreshTokenHashParams{
		RefreshTokenHash: pgtype.Text{
			String: refreshTokenHash,
			Valid:  true,
		},
		ID: user.ID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update refresh token", slog.Any("error", err))
		return PostApiLogin500JSONResponse{
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
