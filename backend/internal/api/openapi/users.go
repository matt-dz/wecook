package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/invite"
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

func (Server) GetApiUser(ctx context.Context, request GetApiUserRequestObject) (GetApiUserResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return GetApiUser500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Get user
	env.Logger.DebugContext(ctx, "get user")
	user, err := env.Database.GetUserById(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "user not found", slog.Any("error", err))
		return GetApiUser404JSONResponse{
			Status:  apiError.UserNotFound.StatusCode(),
			Code:    apiError.UserNotFound.String(),
			Message: "user not found",
			ErrorId: requestID,
		}, nil
	}
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get user", slog.Any("error", err))
		return GetApiUser500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return GetApiUser200JSONResponse{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      Role(user.Role),
	}, nil
}

func (Server) PostApiUserInvite(ctx context.Context,
	request PostApiUserInviteRequestObject,
) (PostApiUserInviteResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)
	userID, err := token.UserIDFromCtx(ctx)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to extract user id from context", slog.Any("error", err))
		return PostApiUserInvite500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Create invite
	const inviteCodeBytes = 16
	env.Logger.DebugContext(ctx, "creating invite code")
	code, err := token.CreateToken(inviteCodeBytes)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create code", slog.Any("error", err))
		return PostApiUserInvite500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	codeHash, err := argon2id.EncodeHash(code, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash code", slog.Any("error", err))
		return PostApiUserInvite500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	inviteID, err := env.Database.CreateInviteCode(ctx, database.CreateInviteCodeParams{
		CodeHash: codeHash,
		InvitedBy: pgtype.Int8{
			Int64: userID,
			Valid: true,
		},
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create invite code", slog.Any("error", err))
		return PostApiUserInvite500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Encode invite
	invite := invite.EncodeInvite(code, inviteID)
	inviteLink := fmt.Sprintf("%s/signup?code=%s",
		strings.TrimRight(env.Get("BASE_URL"), "/"), invite)

	//nolint:lll
	msg := fmt.Sprintf(`Hello!

You have been invited to sign up for WeCook — a platform for creating and sharing recipes. Signup via the invite link below (note the link expires in 8 hours):

%s`, inviteLink)

	// Send invite
	env.Logger.DebugContext(ctx, "sending invite")
	err = env.SMTP.Send([]string{request.Body.Email}, "WeCook Invitation", msg)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to send invite", slog.Any("error", err))
		return PostApiUserInvite500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return PostApiUserInvite204Response{}, nil
}
