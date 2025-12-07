// Package admin contains handlers for the admin endpoints
package admin

import (
	"log/slog"
	"net/http"
	"strconv"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/env"
)

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

	// HandleCreateAdmin(w, r)
}
