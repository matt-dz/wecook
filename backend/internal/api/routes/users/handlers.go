// Package users contains handlers for the user resource.
package users

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	mJson "github.com/matt-dz/wecook/internal/json"
	"github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/password"
	"github.com/matt-dz/wecook/internal/role"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// HandleCreateUser godoc
//
//	@Summary	Create a user.
//	@Tags		User
//
//	@Accept		json
//	@Param		request	body	CreateUserRequest	true	"Create User Request"
//	@Params		cookie header string true "access=..."
//
//	@Success	200	{object}	CreateUserResponse
//	@Failure	409	{object}	apiError.Error	"Status Conflict"
//	@Failure	422	{object}	apiError.Error	"Unprocessible Entity"
//	@Router		/api/admin/user [POST]
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Decode JSON
	var request CreateUserRequest
	env.Logger.DebugContext(ctx, "Reading request body")
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := mJson.DecodeJSON(&request, decoder); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to decode request body", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid request body", requestID)
		return
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to validate request body", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid request body", requestID)
		return
	}

	// Ensure password strength
	env.Logger.DebugContext(ctx, "Validating password")
	if err := password.ValidatePassword(request.Password); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to validate password", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.WeakPassword, err.Error(), requestID) // OK to share the error with client.
		return
	}

	// Hash password
	env.Logger.DebugContext(ctx, "Hashing password")
	hash, err := argon2id.EncodeHash(request.Password, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to hash password", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Create user
	var pgErr *pgconn.PgError
	env.Logger.DebugContext(ctx, "Creating user")
	userID, err := env.Database.CreateUser(ctx, database.CreateUserParams{
		Email:        request.Email,
		PasswordHash: hash,
		FirstName:    request.FirstName,
		LastName:     request.LastName,
	})
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ColumnName == "email" {
		env.Logger.ErrorContext(ctx, "User with email already exists", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.EmailConflict, "email already in use", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to create user", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.DebugContext(ctx, "Writing response")
	resp, err := json.Marshal(CreateUserResponse{UserID: userID})
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to marshal response", slog.Any("error", err))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to write response", slog.Any("error", err))
	}
}

// HandleUserLogin godoc
//
//	@Summary	User login.
//
//	@Tags		User
//
//	@Accept		json
//	@Param		request	body	UserLoginRequest	true	"User Login Request"
//
//	@Success	200
//	@Failure	401	{object}	apiError.Error	"Unauthorized"
//	@Router		/api/login [POST]
func HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Decode JSON
	var request UserLoginRequest
	env.Logger.DebugContext(ctx, "Reading request body")
	defer func() { _ = r.Body.Close() }()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := mJson.DecodeJSON(&request, decoder); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to decode request body", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid request body", requestID)
		return
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(request); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to validate request body", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.BadRequest, "invalid request body", requestID)
		return
	}

	// Retrieve user information
	env.Logger.DebugContext(ctx, "Retrieving user information")
	user, err := env.Database.GetUser(ctx, request.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx,
			"User with email does not exist",
			slog.String("email", request.Email),
			slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "username or password is incorrect", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to retrieve user information", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Decode user password
	env.Logger.DebugContext(ctx, "Decoding user password")
	argonParams, argonSalt, trueHash, err := argon2id.DecodeHash(user.PasswordHash)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to decode password hash", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Hash given password
	env.Logger.DebugContext(ctx, "Hashing given password")
	givenHash := argon2id.EncodeHashWithSalt(request.Password, *argonParams, argonSalt)

	// Comparing passwords
	env.Logger.DebugContext(ctx, "Comparing passwords")
	if subtle.ConstantTimeCompare([]byte(givenHash), trueHash) != 0 {
		env.Logger.ErrorContext(ctx, "Given password is incorrect")
		_ = apiError.EncodeError(w, apiError.InvalidCredentials, "username or password is incorrect", requestID)
		return
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
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Create refresh token
	env.Logger.DebugContext(ctx, "Generating refresh token")
	refreshToken, err := token.NewRefreshToken(user.ID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create refresh token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}
	refreshTokenHash, err := argon2id.EncodeHash(refreshToken, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash refresh token", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
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
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.DebugContext(ctx, "Writing response")
	http.SetCookie(w, token.NewAccessTokenCookie(accessToken, env))
	http.SetCookie(w, token.NewRefreshTokenCookie(refreshToken, env))
}
