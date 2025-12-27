package client

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
)

func (Server) GetApiUsers(ctx context.Context, request GetApiUsersRequestObject) (GetApiUsersResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	var after int64
	if request.Params.After != nil {
		after = *request.Params.After
	}

	var limit int32
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	env.Logger.DebugContext(ctx, "getting users")
	users, err := env.Database.GetUsers(ctx, database.GetUsersParams{
		After: pgtype.Int8{
			Int64: after,
			Valid: request.Params.After != nil,
		},
		Limit: pgtype.Int4{
			Int32: limit,
			Valid: request.Params.Limit != nil,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get users", slog.Any("error", err))
		return GetApiUsers500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	res := GetApiUsers200JSONResponse{
		Users: make([]User, len(users)),
	}
	for idx, user := range users {
		res.Users[idx] = User{
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Id:        user.ID,
			Role:      Role(user.Role),
		}
		res.Cursor = max(res.Cursor, user.ID)
	}

	return res, nil
}
