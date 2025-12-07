package client

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/password"
)

func (Server) PostApiAdmin(ctx context.Context, request PostApiAdminRequestObject) (PostApiAdminResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Ensure password strength
	env.Logger.DebugContext(ctx, "Validating password")
	if err := password.ValidatePassword(request.Body.Password); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to validate password", slog.Any("error", err))

		return PostApiAdmin422JSONResponse{
			Status:  apiError.WeakPassword.StatusCode(),
			Code:    apiError.WeakPassword.String(),
			Message: err.Error(),
			ErrorId: requestID,
		}, nil
	}

	// Hash password
	env.Logger.DebugContext(ctx, "Hashing password")
	hash, err := argon2id.EncodeHash(request.Body.Password, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to hash password", slog.Any("error", err))
		return PostApiAdmin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Create admin
	var pgErr *pgconn.PgError
	env.Logger.DebugContext(ctx, "Creating admin")
	_, err = env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		Email:        string(request.Body.Email),
		PasswordHash: hash,
		FirstName:    request.Body.FirstName,
		LastName:     request.Body.LastName,
	})
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_unique_email" {
		env.Logger.ErrorContext(ctx, "email already in use", slog.Any("error", err))
		return PostApiAdmin409JSONResponse{
			Status:  apiError.WeakPassword.StatusCode(),
			Code:    apiError.WeakPassword.String(),
			Message: "email already in use",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to create admin", slog.Any("error", err))
		return PostApiAdmin500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiAdmin204JSONResponse{}, nil
}

func (Server) PostApiAdminUser(ctx context.Context,
	request PostApiAdminUserRequestObject,
) (PostApiAdminUserResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Ensure password strength
	env.Logger.DebugContext(ctx, "Validating password")
	if err := password.ValidatePassword(request.Body.Password); err != nil {
		env.Logger.ErrorContext(ctx, "Failed to validate password", slog.Any("error", err))
		return PostApiAdminUser409JSONResponse{
			Status:  apiError.WeakPassword.StatusCode(),
			Code:    apiError.WeakPassword.String(),
			Message: "weak password",
			ErrorId: requestID,
		}, nil
	}

	// Hash password
	env.Logger.DebugContext(ctx, "Hashing password")
	passwordHash, err := argon2id.EncodeHash(request.Body.Password, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to hash password", slog.Any("error", err))
		return PostApiAdminUser500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Create admin
	var pgErr *pgconn.PgError
	env.Logger.DebugContext(ctx, "Creating admin")
	_, err = env.Database.CreateAdmin(ctx, database.CreateAdminParams{
		Email:        string(request.Body.Email),
		PasswordHash: passwordHash,
		FirstName:    request.Body.FirstName,
		LastName:     request.Body.LastName,
	})
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_unique_email" {
		env.Logger.ErrorContext(ctx, "email already in use", slog.Any("error", err))
		return PostApiAdminUser422JSONResponse{
			Code:    apiError.EmailConflict.String(),
			Status:  apiError.EmailConflict.StatusCode(),
			Message: "email already in use",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "Failed to create admin", slog.Any("error", err))
		return PostApiAdminUser500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiAdminUser204JSONResponse{}, nil
}
