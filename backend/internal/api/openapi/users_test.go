package client

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/api/token"
	"github.com/matt-dz/wecook/internal/argon2id"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/email"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestGetApiUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    GetApiUsersRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantUsers  int
		wantCursor int64
		wantError  bool
	}{
		{
			name: "successful retrieval with no parameters",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), database.GetUsersParams{
						After: pgtype.Int8{
							Int64: 0,
							Valid: false,
						},
						Limit: pgtype.Int4{
							Int32: 0,
							Valid: false,
						},
					}).
					Return([]database.GetUsersRow{
						{
							ID:        1,
							Email:     "user1@example.com",
							FirstName: "John",
							LastName:  "Doe",
							Role:      database.RoleUser,
						},
						{
							ID:        2,
							Email:     "user2@example.com",
							FirstName: "Jane",
							LastName:  "Smith",
							Role:      database.RoleAdmin,
						},
					}, nil)
			},
			wantStatus: 200,
			wantUsers:  2,
			wantCursor: 2,
			wantError:  false,
		},
		{
			name: "successful retrieval with limit parameter",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{
					Limit: int32Ptr(10),
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), database.GetUsersParams{
						After: pgtype.Int8{
							Int64: 0,
							Valid: false,
						},
						Limit: pgtype.Int4{
							Int32: 10,
							Valid: true,
						},
					}).
					Return([]database.GetUsersRow{
						{
							ID:        1,
							Email:     "user1@example.com",
							FirstName: "John",
							LastName:  "Doe",
							Role:      database.RoleUser,
						},
					}, nil)
			},
			wantStatus: 200,
			wantUsers:  1,
			wantCursor: 1,
			wantError:  false,
		},
		{
			name: "successful retrieval with after parameter",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{
					After: int64Ptr(5),
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), database.GetUsersParams{
						After: pgtype.Int8{
							Int64: 5,
							Valid: true,
						},
						Limit: pgtype.Int4{
							Int32: 0,
							Valid: false,
						},
					}).
					Return([]database.GetUsersRow{
						{
							ID:        6,
							Email:     "user6@example.com",
							FirstName: "Bob",
							LastName:  "Johnson",
							Role:      database.RoleUser,
						},
						{
							ID:        7,
							Email:     "user7@example.com",
							FirstName: "Alice",
							LastName:  "Williams",
							Role:      database.RoleUser,
						},
					}, nil)
			},
			wantStatus: 200,
			wantUsers:  2,
			wantCursor: 7,
			wantError:  false,
		},
		{
			name: "successful retrieval with both after and limit parameters",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{
					After: int64Ptr(10),
					Limit: int32Ptr(5),
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), database.GetUsersParams{
						After: pgtype.Int8{
							Int64: 10,
							Valid: true,
						},
						Limit: pgtype.Int4{
							Int32: 5,
							Valid: true,
						},
					}).
					Return([]database.GetUsersRow{
						{
							ID:        11,
							Email:     "user11@example.com",
							FirstName: "Charlie",
							LastName:  "Brown",
							Role:      database.RoleAdmin,
						},
						{
							ID:        12,
							Email:     "user12@example.com",
							FirstName: "David",
							LastName:  "Miller",
							Role:      database.RoleUser,
						},
						{
							ID:        13,
							Email:     "user13@example.com",
							FirstName: "Eve",
							LastName:  "Davis",
							Role:      database.RoleUser,
						},
					}, nil)
			},
			wantStatus: 200,
			wantUsers:  3,
			wantCursor: 13,
			wantError:  false,
		},
		{
			name: "empty result set",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{
					After: int64Ptr(1000),
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), database.GetUsersParams{
						After: pgtype.Int8{
							Int64: 1000,
							Valid: true,
						},
						Limit: pgtype.Int4{
							Int32: 0,
							Valid: false,
						},
					}).
					Return([]database.GetUsersRow{}, nil)
			},
			wantStatus: 200,
			wantUsers:  0,
			wantCursor: 0,
			wantError:  false,
		},
		{
			name: "database error",
			request: GetApiUsersRequestObject{
				Params: GetApiUsersParams{},
			},
			setup: func() {
				mockDB.EXPECT().
					GetUsers(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			resp, err := server.GetApiUsers(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiUsers() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case GetApiUsers200JSONResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if len(v.Users) != tt.wantUsers {
					t.Errorf("expected %d users, got %d", tt.wantUsers, len(v.Users))
				}
				if v.Cursor != tt.wantCursor {
					t.Errorf("expected cursor %d, got %d", tt.wantCursor, v.Cursor)
				}
			case GetApiUsers500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestGetApiUsers_UserFieldMapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	mockDB.EXPECT().
		GetUsers(gomock.Any(), gomock.Any()).
		Return([]database.GetUsersRow{
			{
				ID:        123,
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Role:      database.RoleAdmin,
			},
		}, nil)

	ctx := context.Background()
	ctx = requestid.InjectRequestID(ctx, 12345)
	ctx = env.WithCtx(ctx, &env.Env{
		Logger: log.NullLogger(),
		Database: &database.Database{
			Querier: mockDB,
		},
	})

	request := GetApiUsersRequestObject{
		Params: GetApiUsersParams{},
	}

	resp, err := server.GetApiUsers(ctx, request)
	if err != nil {
		t.Fatalf("GetApiUsers() error = %v", err)
	}

	successResp, ok := resp.(GetApiUsers200JSONResponse)
	if !ok {
		t.Fatalf("expected GetApiUsers200JSONResponse, got %T", resp)
	}

	if len(successResp.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(successResp.Users))
	}

	user := successResp.Users[0]
	if user.Id != 123 {
		t.Errorf("expected user ID 123, got %d", user.Id)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", user.Email)
	}
	if user.FirstName != "Test" {
		t.Errorf("expected first name 'Test', got %s", user.FirstName)
	}
	if user.LastName != "User" {
		t.Errorf("expected last name 'User', got %s", user.LastName)
	}
	if user.Role != RoleAdmin {
		t.Errorf("expected role Admin, got %s", user.Role)
	}
}

func TestGetApiUsers_CursorCalculation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		users      []database.GetUsersRow
		wantCursor int64
	}{
		{
			name: "cursor is max ID when users returned in ascending order",
			users: []database.GetUsersRow{
				{ID: 1, Email: "user1@example.com", FirstName: "User", LastName: "One", Role: database.RoleUser},
				{ID: 5, Email: "user5@example.com", FirstName: "User", LastName: "Five", Role: database.RoleUser},
				{ID: 10, Email: "user10@example.com", FirstName: "User", LastName: "Ten", Role: database.RoleUser},
			},
			wantCursor: 10,
		},
		{
			name: "cursor is max ID when users returned in descending order",
			users: []database.GetUsersRow{
				{ID: 10, Email: "user10@example.com", FirstName: "User", LastName: "Ten", Role: database.RoleUser},
				{ID: 5, Email: "user5@example.com", FirstName: "User", LastName: "Five", Role: database.RoleUser},
				{ID: 1, Email: "user1@example.com", FirstName: "User", LastName: "One", Role: database.RoleUser},
			},
			wantCursor: 10,
		},
		{
			name: "cursor is max ID when users returned in random order",
			users: []database.GetUsersRow{
				{ID: 5, Email: "user5@example.com", FirstName: "User", LastName: "Five", Role: database.RoleUser},
				{ID: 15, Email: "user15@example.com", FirstName: "User", LastName: "Fifteen", Role: database.RoleUser},
				{ID: 3, Email: "user3@example.com", FirstName: "User", LastName: "Three", Role: database.RoleUser},
				{ID: 10, Email: "user10@example.com", FirstName: "User", LastName: "Ten", Role: database.RoleUser},
			},
			wantCursor: 15,
		},
		{
			name:       "cursor is 0 when no users",
			users:      []database.GetUsersRow{},
			wantCursor: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().
				GetUsers(gomock.Any(), gomock.Any()).
				Return(tt.users, nil)

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			request := GetApiUsersRequestObject{
				Params: GetApiUsersParams{},
			}

			resp, err := server.GetApiUsers(ctx, request)
			if err != nil {
				t.Fatalf("GetApiUsers() error = %v", err)
			}

			successResp, ok := resp.(GetApiUsers200JSONResponse)
			if !ok {
				t.Fatalf("expected GetApiUsers200JSONResponse, got %T", resp)
			}

			if successResp.Cursor != tt.wantCursor {
				t.Errorf("expected cursor %d, got %d", tt.wantCursor, successResp.Cursor)
			}
		})
	}
}

func TestGetApiUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		setup      func() context.Context
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful retrieval - user role",
			setup: func() context.Context {
				mockDB.EXPECT().
					GetUserById(gomock.Any(), int64(123)).
					Return(database.GetUserByIdRow{
						ID:        123,
						Email:     "user@example.com",
						FirstName: "John",
						LastName:  "Doe",
						Role:      database.RoleUser,
					}, nil)

				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = token.UserIDWithCtx(ctx, 123)
				ctx = env.WithCtx(ctx, &env.Env{
					Logger: log.NullLogger(),
					Database: &database.Database{
						Querier: mockDB,
					},
				})
				return ctx
			},
			wantStatus: 200,
			wantError:  false,
		},
		{
			name: "successful retrieval - admin role",
			setup: func() context.Context {
				mockDB.EXPECT().
					GetUserById(gomock.Any(), int64(456)).
					Return(database.GetUserByIdRow{
						ID:        456,
						Email:     "admin@example.com",
						FirstName: "Jane",
						LastName:  "Smith",
						Role:      database.RoleAdmin,
					}, nil)

				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = token.UserIDWithCtx(ctx, 456)
				ctx = env.WithCtx(ctx, &env.Env{
					Logger: log.NullLogger(),
					Database: &database.Database{
						Querier: mockDB,
					},
				})
				return ctx
			},
			wantStatus: 200,
			wantError:  false,
		},
		{
			name: "missing user id in context",
			setup: func() context.Context {
				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = env.WithCtx(ctx, &env.Env{
					Logger: log.NullLogger(),
					Database: &database.Database{
						Querier: mockDB,
					},
				})
				return ctx
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "user not found",
			setup: func() context.Context {
				mockDB.EXPECT().
					GetUserById(gomock.Any(), int64(999)).
					Return(database.GetUserByIdRow{}, pgx.ErrNoRows)

				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = token.UserIDWithCtx(ctx, 999)
				ctx = env.WithCtx(ctx, &env.Env{
					Logger: log.NullLogger(),
					Database: &database.Database{
						Querier: mockDB,
					},
				})
				return ctx
			},
			wantStatus: 404,
			wantCode:   apiError.UserNotFound.String(),
			wantError:  false,
		},
		{
			name: "database error",
			setup: func() context.Context {
				mockDB.EXPECT().
					GetUserById(gomock.Any(), int64(123)).
					Return(database.GetUserByIdRow{}, errors.New("database connection error"))

				ctx := context.Background()
				ctx = requestid.InjectRequestID(ctx, 12345)
				ctx = token.UserIDWithCtx(ctx, 123)
				ctx = env.WithCtx(ctx, &env.Env{
					Logger: log.NullLogger(),
					Database: &database.Database{
						Querier: mockDB,
					},
				})
				return ctx
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			request := GetApiUserRequestObject{}

			resp, err := server.GetApiUser(ctx, request)
			if (err != nil) != tt.wantError {
				t.Errorf("GetApiUser() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case GetApiUser200JSONResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
			case GetApiUser404JSONResponse:
				if tt.wantStatus != 404 {
					t.Errorf("expected status %d, got 404", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case GetApiUser500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestGetApiUser_FieldMapping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	tests := []struct {
		name     string
		userID   int64
		dbUser   database.GetUserByIdRow
		wantUser func(t *testing.T, user GetApiUser200JSONResponse)
	}{
		{
			name:   "user role with all fields",
			userID: 123,
			dbUser: database.GetUserByIdRow{
				ID:        123,
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Role:      database.RoleUser,
			},
			wantUser: func(t *testing.T, user GetApiUser200JSONResponse) {
				if user.Id != 123 {
					t.Errorf("expected user ID 123, got %d", user.Id)
				}
				if user.Email != "user@example.com" {
					t.Errorf("expected email 'user@example.com', got %s", user.Email)
				}
				if user.FirstName != "John" {
					t.Errorf("expected first name 'John', got %s", user.FirstName)
				}
				if user.LastName != "Doe" {
					t.Errorf("expected last name 'Doe', got %s", user.LastName)
				}
				if user.Role != RoleUser {
					t.Errorf("expected role User, got %s", user.Role)
				}
			},
		},
		{
			name:   "admin role with all fields",
			userID: 456,
			dbUser: database.GetUserByIdRow{
				ID:        456,
				Email:     "admin@example.com",
				FirstName: "Jane",
				LastName:  "Smith",
				Role:      database.RoleAdmin,
			},
			wantUser: func(t *testing.T, user GetApiUser200JSONResponse) {
				if user.Id != 456 {
					t.Errorf("expected user ID 456, got %d", user.Id)
				}
				if user.Email != "admin@example.com" {
					t.Errorf("expected email 'admin@example.com', got %s", user.Email)
				}
				if user.FirstName != "Jane" {
					t.Errorf("expected first name 'Jane', got %s", user.FirstName)
				}
				if user.LastName != "Smith" {
					t.Errorf("expected last name 'Smith', got %s", user.LastName)
				}
				if user.Role != RoleAdmin {
					t.Errorf("expected role Admin, got %s", user.Role)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().
				GetUserById(gomock.Any(), tt.userID).
				Return(tt.dbUser, nil)

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			ctx = token.UserIDWithCtx(ctx, tt.userID)
			ctx = env.WithCtx(ctx, &env.Env{
				Logger: log.NullLogger(),
				Database: &database.Database{
					Querier: mockDB,
				},
			})

			request := GetApiUserRequestObject{}
			resp, err := server.GetApiUser(ctx, request)
			if err != nil {
				t.Fatalf("GetApiUser() error = %v", err)
			}

			successResp, ok := resp.(GetApiUser200JSONResponse)
			if !ok {
				t.Fatalf("expected GetApiUser200JSONResponse, got %T", resp)
			}

			tt.wantUser(t, successResp)
		})
	}
}

func TestPostApiUserInvite(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	mockSMTP := email.NewMockSender(ctrl)
	server := NewServer()

	tests := []struct {
		name       string
		request    PostApiUserInviteRequestObject
		setup      func(ctx context.Context) context.Context
		dbSetup    func()
		smtpSetup  func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful invite creation and email sending",
			request: PostApiUserInviteRequestObject{
				Body: &InviteUserRequest{
					Email: "newuser@example.com",
				},
			},
			setup: func(ctx context.Context) context.Context {
				return token.UserIDWithCtx(ctx, 123)
			},
			dbSetup: func() {
				mockDB.EXPECT().
					CreateInviteCode(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, params database.CreateInviteCodeParams) (int64, error) {
						if !params.InvitedBy.Valid || params.InvitedBy.Int64 != 123 {
							t.Errorf("expected invited_by to be 123, got %v", params.InvitedBy)
						}
						if params.CodeHash == "" {
							t.Error("expected non-empty code hash")
						}
						return int64(456), nil
					})
			},
			smtpSetup: func() {
				mockSMTP.EXPECT().
					Send(gomock.Eq([]string{"newuser@example.com"}), gomock.Eq("WeCook Invitation"), gomock.Any()).
					DoAndReturn(func(to []string, subject, body string) error {
						if !strings.Contains(body, "http://localhost:5173/signup?code=") {
							t.Errorf("expected invite link in email body, got: %s", body)
						}
						return nil
					})
			},
			wantStatus: 204,
			wantError:  false,
		},
		{
			name: "missing user ID in context",
			request: PostApiUserInviteRequestObject{
				Body: &InviteUserRequest{
					Email: "newuser@example.com",
				},
			},
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			dbSetup:    func() {},
			smtpSetup:  func() {},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "database error creating invite code",
			request: PostApiUserInviteRequestObject{
				Body: &InviteUserRequest{
					Email: "newuser@example.com",
				},
			},
			setup: func(ctx context.Context) context.Context {
				return token.UserIDWithCtx(ctx, 123)
			},
			dbSetup: func() {
				mockDB.EXPECT().
					CreateInviteCode(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database connection error"))
			},
			smtpSetup:  func() {},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "email sending error",
			request: PostApiUserInviteRequestObject{
				Body: &InviteUserRequest{
					Email: "newuser@example.com",
				},
			},
			setup: func(ctx context.Context) context.Context {
				return token.UserIDWithCtx(ctx, 123)
			},
			dbSetup: func() {
				mockDB.EXPECT().
					CreateInviteCode(gomock.Any(), gomock.Any()).
					Return(int64(456), nil)
			},
			smtpSetup: func() {
				mockSMTP.EXPECT().
					Send(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("SMTP connection failed"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.dbSetup()
			tt.smtpSetup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			ctx = tt.setup(ctx)
			e := env.New(map[string]string{
				"BASE_URL": "http://localhost:5173",
			})
			e.Logger = log.NullLogger()
			e.Database = mockDB
			e.SMTP = mockSMTP
			ctx = env.WithCtx(ctx, e)

			resp, err := server.PostApiUserInvite(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiUserInvite() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case PostApiUserInvite204Response:
				if tt.wantStatus != 204 {
					t.Errorf("expected status %d, got 204", tt.wantStatus)
				}
			case PostApiUserInvite500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if tt.wantCode != "" && v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestPostApiSignup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	// Create a valid invite code and hash
	validInviteCode := "456$test-code-1234567890"
	inviteCodeOnly := "test-code-1234567890"
	validCodeHash, err := argon2id.EncodeHash(inviteCodeOnly, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to create test code hash: %v", err)
	}

	testPassword := "ValidP@ssw0rd123!"

	tests := []struct {
		name       string
		request    PostApiSignupRequestObject
		setup      func()
		wantStatus int
		wantCode   string
		wantError  bool
	}{
		{
			name: "successful signup",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(123), nil)

				mockDB.EXPECT().
					RedeemInvitationCode(gomock.Any(), int64(456)).
					Return(int64(1), nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantStatus: 200,
			wantError:  false,
		},
		{
			name: "missing invite code",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:     openapi_types.Email("newuser@example.com"),
					FirstName: "John",
					LastName:  "Doe",
					Password:  testPassword,
				},
			},
			setup:      func() {},
			wantStatus: 400,
			wantCode:   apiError.BadRequest.String(),
			wantError:  false,
		},
		{
			name: "invalid invite code format",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: stringPtr("invalid-no-delimiter"),
				},
			},
			setup:      func() {},
			wantStatus: 422,
			wantCode:   apiError.InvalidInviteCode.String(),
			wantError:  false,
		},
		{
			name: "invite code not found in database",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return("", pgx.ErrNoRows)
			},
			wantStatus: 422,
			wantCode:   apiError.InvalidInviteCode.String(),
			wantError:  false,
		},
		{
			name: "database error getting invitation code",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return("", errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "invalid invite code - hash mismatch",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: stringPtr("456$wrong-code-here"),
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)
			},
			wantStatus: 422,
			wantCode:   apiError.InvalidInviteCode.String(),
			wantError:  false,
		},
		{
			name: "weak password",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   "weak",
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)
			},
			wantStatus: 422,
			wantCode:   apiError.InvalidPassword.String(),
			wantError:  false,
		},
		{
			name: "duplicate email",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("existing@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				pgErr := &pgconn.PgError{
					Code: "23505",
				}
				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(0), pgErr)
			},
			wantStatus: 422,
			wantCode:   apiError.EmailConflict.String(),
			wantError:  false,
		},
		{
			name: "database error creating user",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "invite code already used - redeem returns 0 rows",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(123), nil)

				mockDB.EXPECT().
					RedeemInvitationCode(gomock.Any(), int64(456)).
					Return(int64(0), nil)
			},
			wantStatus: 422,
			wantCode:   apiError.InvalidInviteCode.String(),
			wantError:  false,
		},
		{
			name: "database error redeeming invitation code",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(123), nil)

				mockDB.EXPECT().
					RedeemInvitationCode(gomock.Any(), int64(456)).
					Return(int64(0), errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
		{
			name: "error updating user refresh token",
			request: PostApiSignupRequestObject{
				Body: &SignupRequest{
					Email:      openapi_types.Email("newuser@example.com"),
					FirstName:  "John",
					LastName:   "Doe",
					Password:   testPassword,
					InviteCode: &validInviteCode,
				},
			},
			setup: func() {
				mockDB.EXPECT().
					GetInvitationCode(gomock.Any(), int64(456)).
					Return(validCodeHash, nil)

				mockDB.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(int64(123), nil)

				mockDB.EXPECT().
					RedeemInvitationCode(gomock.Any(), int64(456)).
					Return(int64(1), nil)

				mockDB.EXPECT().
					UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
					Return(errors.New("database connection error"))
			},
			wantStatus: 500,
			wantCode:   apiError.InternalServerError.String(),
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			ctx := context.Background()
			ctx = requestid.InjectRequestID(ctx, 12345)
			e := env.New(map[string]string{
				"APP_SECRET": "test-secret-key-for-jwt-signing-at-least-32-characters-long",
				"ENV":        "PROD",
			})
			e.Logger = log.NullLogger()
			e.Database = mockDB
			ctx = env.WithCtx(ctx, e)

			resp, err := server.PostApiSignup(ctx, tt.request)
			if (err != nil) != tt.wantError {
				t.Errorf("PostApiSignup() error = %v, wantError %v", err, tt.wantError)
				return
			}

			switch v := resp.(type) {
			case loginSuccessResponse:
				if tt.wantStatus != 200 {
					t.Errorf("expected status %d, got 200", tt.wantStatus)
				}
				if v.accessCookie == nil {
					t.Error("expected access cookie, got nil")
				}
				if v.refreshCookie == nil {
					t.Error("expected refresh cookie, got nil")
				}
				if v.body.AccessToken == "" {
					t.Error("expected access token in body, got empty string")
				}
			case PostApiSignup400JSONResponse:
				if tt.wantStatus != 400 {
					t.Errorf("expected status %d, got 400", tt.wantStatus)
				}
				if tt.wantCode != "" && v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiSignup422JSONResponse:
				if tt.wantStatus != 422 {
					t.Errorf("expected status %d, got 422", tt.wantStatus)
				}
				if tt.wantCode != "" && v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			case PostApiSignup500JSONResponse:
				if tt.wantStatus != 500 {
					t.Errorf("expected status %d, got 500", tt.wantStatus)
				}
				if tt.wantCode != "" && v.Code != tt.wantCode {
					t.Errorf("expected code %s, got %s", tt.wantCode, v.Code)
				}
			default:
				t.Errorf("unexpected response type: %T", v)
			}
		})
	}
}

func TestPostApiSignup_ParameterValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := database.NewMockQuerier(ctrl)
	server := NewServer()

	validInviteCode := "456$test-code-1234567890"
	inviteCodeOnly := "test-code-1234567890"
	validCodeHash, err := argon2id.EncodeHash(inviteCodeOnly, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("failed to create test code hash: %v", err)
	}

	testPassword := "ValidP@ssw0rd123!"

	mockDB.EXPECT().
		GetInvitationCode(gomock.Any(), int64(456)).
		Return(validCodeHash, nil)

	mockDB.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, params database.CreateUserParams) (int64, error) {
			if params.Email != "test@example.com" {
				t.Errorf("expected email 'test@example.com', got %s", params.Email)
			}
			if params.FirstName != "John" {
				t.Errorf("expected first name 'John', got %s", params.FirstName)
			}
			if params.LastName != "Doe" {
				t.Errorf("expected last name 'Doe', got %s", params.LastName)
			}
			if params.PasswordHash == "" {
				t.Error("expected non-empty password hash")
			}
			return int64(123), nil
		})

	mockDB.EXPECT().
		RedeemInvitationCode(gomock.Any(), int64(456)).
		Return(int64(1), nil)

	mockDB.EXPECT().
		UpdateUserRefreshTokenHash(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, params database.UpdateUserRefreshTokenHashParams) error {
			if params.ID != 123 {
				t.Errorf("expected user ID 123, got %d", params.ID)
			}
			if !params.RefreshTokenHash.Valid {
				t.Error("expected valid refresh token hash")
			}
			if params.RefreshTokenHash.String == "" {
				t.Error("expected non-empty refresh token hash")
			}
			return nil
		})

	ctx := context.Background()
	ctx = requestid.InjectRequestID(ctx, 12345)
	e := env.New(map[string]string{
		"APP_SECRET": "test-secret-key-for-jwt-signing-at-least-32-characters-long",
		"ENV":        "PROD",
	})
	e.Logger = log.NullLogger()
	e.Database = mockDB
	ctx = env.WithCtx(ctx, e)

	request := PostApiSignupRequestObject{
		Body: &SignupRequest{
			Email:      openapi_types.Email("test@example.com"),
			FirstName:  "John",
			LastName:   "Doe",
			Password:   testPassword,
			InviteCode: &validInviteCode,
		},
	}

	resp, err := server.PostApiSignup(ctx, request)
	if err != nil {
		t.Fatalf("PostApiSignup() error = %v", err)
	}

	_, ok := resp.(loginSuccessResponse)
	if !ok {
		t.Fatalf("expected loginSuccessResponse, got %T", resp)
	}
}
