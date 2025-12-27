package client

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/invite"
	mJwt "github.com/matt-dz/wecook/internal/jwt"
	"github.com/matt-dz/wecook/internal/password"
	"github.com/matt-dz/wecook/internal/role"
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
	invite := invite.EncodeInvite(inviteID, code)
	inviteLink := fmt.Sprintf("%s/signup?code=%s",
		strings.TrimRight(env.Get("BASE_URL"), "/"), invite)

	//nolint:lll
	msg := fmt.Sprintf(`Hello!

You have been invited to sign up for WeCook — a platform for creating and sharing recipes. Signup via the invite link below (note the link expires in 8 hours):

%s`, inviteLink)

	// Send invite
	env.Logger.DebugContext(ctx, "sending invite")
	err = env.SMTP.Send([]string{string(request.Body.Email)}, "WeCook Invitation", msg)
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

func (Server) PostApiSignup(ctx context.Context,
	request PostApiSignupRequestObject,
) (PostApiSignupResponseObject, error) {
	env := env.EnvFromCtx(ctx)
	requestID := strconv.FormatUint(requestid.ExtractRequestID(ctx), 10)

	// TODO: make invite-only configurable
	if request.Body.InviteCode == nil {
		env.Logger.ErrorContext(ctx, "invite code not provided")
		return PostApiSignup400JSONResponse{
			Status:  apiError.BadRequest.StatusCode(),
			Code:    apiError.BadRequest.String(),
			Message: "missing invite code",
			ErrorId: requestID,
		}, nil
	}

	// Decode invite
	env.Logger.DebugContext(ctx, "decoding invite")
	inviteid, code, err := invite.DecodeInvite(*request.Body.InviteCode)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to decode invite code", slog.Any("error", err))
		return PostApiSignup422JSONResponse{
			Status:  apiError.InvalidInviteCode.StatusCode(),
			Code:    apiError.InvalidInviteCode.String(),
			Message: "invalid invite code",
			ErrorId: requestID,
		}, nil
	}

	// Retrieve code from db
	env.Logger.DebugContext(ctx, "getting invitation code")
	encodedGroundHash, err := env.Database.GetInvitationCode(ctx, inviteid)
	if errors.Is(err, pgx.ErrNoRows) {
		env.Logger.ErrorContext(ctx, "invitation code does not exist", slog.Any("error", err))
		return PostApiSignup422JSONResponse{
			Status:  apiError.InvalidInviteCode.StatusCode(),
			Code:    apiError.InvalidInviteCode.String(),
			Message: "invalid invite code",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to get invitation code", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Decode ground hash
	env.Logger.DebugContext(ctx, "decoding ground hash code")
	p, salt, groundHash, err := argon2id.DecodeHash(encodedGroundHash)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to decode hash", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Encode given hash
	env.Logger.DebugContext(ctx, "encoding given hash")
	givenCodeHash := argon2id.HashWithSalt(code, *p, salt)

	// Compare hashes
	env.Logger.DebugContext(ctx, "comparing hashes")
	if subtle.ConstantTimeCompare(givenCodeHash, groundHash) == 0 {
		env.Logger.ErrorContext(ctx, "codes do not match")
		return PostApiSignup422JSONResponse{
			Status:  apiError.InvalidInviteCode.StatusCode(),
			Code:    apiError.InvalidInviteCode.String(),
			Message: "invalid invite code",
			ErrorId: requestID,
		}, nil
	}

	// Ensure password strength
	env.Logger.DebugContext(ctx, "validating password")
	if err := password.ValidatePassword(request.Body.Password); err != nil {
		env.Logger.ErrorContext(ctx, "invalid password", slog.Any("error", err))
		return PostApiSignup422JSONResponse{
			Status:  apiError.InvalidPassword.StatusCode(),
			Code:    apiError.InvalidPassword.String(),
			Message: err.Error(),
			ErrorId: requestID,
		}, nil
	}

	// Hash password
	env.Logger.DebugContext(ctx, "hashing password")
	passwordHash, err := argon2id.EncodeHash(request.Body.Password, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to hash password", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Create user
	env.Logger.DebugContext(ctx, "creating user")
	userID, err := env.Database.CreateUser(ctx, database.CreateUserParams{
		Email:        string(request.Body.Email),
		FirstName:    request.Body.FirstName,
		LastName:     request.Body.LastName,
		PasswordHash: passwordHash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		env.Logger.ErrorContext(ctx, "user with email already exists", slog.Any("error", err))
		return PostApiSignup422JSONResponse{
			Status:  apiError.EmailConflict.StatusCode(),
			Code:    apiError.EmailConflict.String(),
			Message: "email already in use",
			ErrorId: requestID,
		}, nil
	} else if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create user", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	// Redeem invitation code
	env.Logger.DebugContext(ctx, "redeem invite code")
	rows, err := env.Database.RedeemInvitationCode(ctx, inviteid)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to redeem invite code", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	} else if rows == 0 {
		// Rare edge case where code becomes invalid during signup.
		// Good to process it otherwise the db would become inconsistent
		// and the invite code would never have a valid used_at field.
		env.Logger.ErrorContext(ctx, "failed to redeem invite code; it may have expired", slog.Any("error", err))
		return PostApiSignup422JSONResponse{
			Status:  apiError.InvalidInviteCode.StatusCode(),
			Code:    apiError.InvalidInviteCode.String(),
			Message: "invalid invite code",
			ErrorId: requestID,
		}, nil
	}

	// Create tokens
	env.Logger.DebugContext(ctx, "creating user tokens")
	refreshToken, err := token.NewRefreshToken(userID)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create refresh token")
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	refreshTokenHash, err := argon2id.EncodeHash(refreshToken, argon2id.DefaultParams)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to encode hash")
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}
	accessToken, err := token.NewAccessToken(mJwt.JWTParams{
		Role:   role.RoleUser,
		UserID: fmt.Sprintf("%d", userID),
	}, env)
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to create access token", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	env.Logger.DebugContext(ctx, "uploading refresh token")
	err = env.Database.UpdateUserRefreshTokenHash(ctx, database.UpdateUserRefreshTokenHashParams{
		RefreshTokenHash: pgtype.Text{
			String: refreshTokenHash,
			Valid:  true,
		},
		ID: userID,
	})
	if err != nil {
		env.Logger.ErrorContext(ctx, "failed to upload refresh token", slog.Any("error", err))
		return PostApiSignup500JSONResponse{
			Status:  apiError.InternalServerError.StatusCode(),
			Code:    apiError.InternalServerError.String(),
			Message: "Internal Server Error",
			ErrorId: requestID,
		}, nil
	}

	return loginSuccessResponse{
		accessCookie:  token.NewAccessTokenCookie(accessToken, env.IsProd()),
		refreshCookie: token.NewRefreshTokenCookie(refreshToken, env.IsProd()),
		body: LoginResponse{
			AccessToken: accessToken,
		},
	}, nil
}
