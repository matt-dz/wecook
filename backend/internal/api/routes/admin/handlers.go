// Package admin contains handlers for the admin endpoints
package admin

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	mJson "github.com/matt-dz/wecook/internal/json"
	"github.com/matt-dz/wecook/internal/password"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
)

// HandleCreateAdmin godoc
//
//	@Summary	Create an admin.
//	@Tags		Admin
//
//	@Accept		json
//	@Param		request	body	CreateAdminRequest	true	"Create Admin Request"
//	@Params		cookie header string true "access=..."
//
//	@Success	200	{object}	CreateAdminResponse
//	@Failure	409	{object}	apiError.Error	"Status Conflict"
//	@Failure	422	{object}	apiError.Error	"Unprocessible Entity"
//	@Router		/api/admin [POST]
func HandleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Decode JSON
	var request CreateAdminRequest
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

	// Create admin
	var pgErr *pgconn.PgError
	env.Logger.DebugContext(ctx, "Creating admin")
	userID, err := env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		Email:        request.Email,
		PasswordHash: hash,
		FirstName:    request.FirstName,
		LastName:     request.LastName,
	})
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ColumnName == "email" {
		env.Logger.ErrorContext(ctx, "Admin with email already exists", slog.Any("error", err))
		_ = apiError.EncodeError(w, apiError.EmailConflict, "email already in use", requestID)
		return
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to create admin", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	// Write response
	env.Logger.DebugContext(ctx, "Writing response")
	resp, err := json.Marshal(CreateAdminResponse{UserID: userID})
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to marshal response", slog.Any("error", err))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to write response", slog.Any("error", err))
	}
}

// HandleAdminSetup godoc
//
//	@Summary		Setup the first admin.
//	@Description	This endpoint should only be hit once on system startup.
//	  There must be no admins in the system for this call to succeed.
//	@Tags			Admin
//
//	@Accept			json
//	@Param			request	body	CreateAdminRequest	true	"Create Admin Request"
//	@Params			cookie header string true "access=..."
//
//	@Success		200	{object}	CreateAdminResponse
//	@Failure		409	{object}	apiError.Error	"Status Conflict"
//	@Failure		422	{object}	apiError.Error	"Unprocessible Entity"
//	@Router			/api/setup/admin [POST]
func HandleAdminSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	adminCount, err := env.Database.GetAdminCount(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to get admin count", slog.Any("error", err))
		_ = apiError.EncodeInternalError(w, requestID)
		return
	}

	if adminCount != 0 {
		env.Logger.ErrorContext(ctx, "An admin already exists", slog.Int64("count", adminCount))
		_ = apiError.EncodeError(w, apiError.AdminAlreadySetup, "an admin has already been setup", requestID)
		return
	}

	HandleCreateAdmin(w, r)
}
