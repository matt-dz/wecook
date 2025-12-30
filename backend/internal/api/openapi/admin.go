package client

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/config"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
)

func (Server) PatchApiPreferences(ctx context.Context,
	request PatchApiPreferencesRequestObject) (
	PatchApiPreferencesResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	updateParams := database.UpdatePreferencesParams{
		ID: config.PreferenceID,
	}
	if request.Body.AllowPublicSignup != nil {
		updateParams.UpdateAllowPublicSignup = pgtype.Bool{
			Bool:  true,
			Valid: true,
		}
		updateParams.AllowPublicSignup = pgtype.Bool{
			Bool:  *request.Body.AllowPublicSignup,
			Valid: true,
		}
	}

	// Update preferences
	env.Logger.DebugContext(ctx, "updating preferences")
	prefs, err := env.Database.UpdatePreferences(ctx, updateParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to update preferences", slog.Any("error", err))
		return PatchApiPreferences500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PatchApiPreferences200JSONResponse{
		AllowPublicSignup: prefs.AllowPublicSignup,
	}, nil
}

func (Server) GetApiPreferences(ctx context.Context,
	request GetApiPreferencesRequestObject) (
	GetApiPreferencesResponseObject, error,
) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// Get GetApiPreferences
	env.Logger.DebugContext(ctx, "getting preferences")
	prefs, err := env.Database.GetPreferences(ctx, config.PreferenceID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get preferences", slog.Any("error", err))
		return GetApiPreferences500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return GetApiPreferences200JSONResponse{
		AllowPublicSignup: prefs.AllowPublicSignup,
	}, nil
}
