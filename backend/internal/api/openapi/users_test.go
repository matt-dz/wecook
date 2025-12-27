package client

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"

	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/database"
	"github.com/matt-dz/wecook/internal/dbmock"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/log"
)

func TestGetApiUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbmock.NewMockQuerier(ctrl)
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
			ctx = env.WithCtx(ctx, env.New(
				log.NullLogger(),
				&database.Database{
					Querier: mockDB,
				},
				nil,
				nil,
				nil,
			))

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

	mockDB := dbmock.NewMockQuerier(ctrl)
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
	ctx = env.WithCtx(ctx, env.New(
		log.NullLogger(),
		&database.Database{
			Querier: mockDB,
		},
		nil,
		nil,
		nil,
	))

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

	mockDB := dbmock.NewMockQuerier(ctrl)
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
			ctx = env.WithCtx(ctx, env.New(
				log.NullLogger(),
				&database.Database{
					Querier: mockDB,
				},
				nil,
				nil,
				nil,
			))

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
