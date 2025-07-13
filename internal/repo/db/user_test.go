package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestRepository_ListUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	page := 1
	size := 10
	filters := map[string]any{"is_active": true}
	testUsers := []*md.User{
		{
			ID:              uuid.New(),
			Name:            "User 1",
			Email:           "user1@example.com",
			Avatar:          "avatar1.jpg",
			IsActive:        true,
			IsEmailVerified: true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              uuid.New(),
			Name:            "User 2",
			Email:           "user2@example.com",
			Avatar:          "avatar2.jpg",
			IsActive:        true,
			IsEmailVerified: false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	tests := []struct {
		name        string
		page        int
		size        int
		filters     map[string]any
		mock        func()
		expected    *dto.PaginatedUserResponse
		expectedErr error
	}{
		{
			name:    "Success",
			page:    page,
			size:    size,
			filters: filters,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userSelectQ)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "avatar",
					"is_active", "is_email_verified", "created_at", "updated_at",
				})
				for _, user := range testUsers {
					rows.AddRow(
						user.ID, user.Name, user.Email, user.Avatar,
						user.IsActive, user.IsEmailVerified, user.CreatedAt, user.UpdatedAt,
					)
				}
				mock.ExpectQuery(regexp.QuoteMeta(userListQ)).
					WithArgs(size, (page-1)*size).
					WillReturnRows(rows)
			},
			expected: &dto.PaginatedUserResponse{
				Data:        testUsers,
				Count:       15,
				TotalPages:  2,
				CurrentPage: page,
				HasNextPage: true,
			},
			expectedErr: nil,
		},
		{
			name:    "CountQueryError",
			page:    page,
			size:    size,
			filters: filters,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userSelectQ)).
					WillReturnError(errors.New("count query error"))
			},
			expected:    nil,
			expectedErr: errors.New("count query error"),
		},
		{
			name:    "ListQueryError",
			page:    page,
			size:    size,
			filters: filters,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userSelectQ)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))
				mock.ExpectQuery(regexp.QuoteMeta(userListQ)).
					WithArgs(size, (page-1)*size).
					WillReturnError(errors.New("list query error"))
			},
			expected:    nil,
			expectedErr: errors.New("list query error"),
		},
		{
			name:    "ScanError",
			page:    page,
			size:    size,
			filters: filters,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userSelectQ)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "avatar",
					"is_active", "is_email_verified", "created_at", "updated_at",
				}).AddRow("invalid-uuid", "User 1", "user1@example.com", nil, true, true, time.Now(), time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(userListQ)).
					WithArgs(size, (page-1)*size).
					WillReturnRows(rows)
			},
			expected:    nil,
			expectedErr: errors.New("Scan: invalid UUID length"),
		},
		{
			name:    "EmptyResult",
			page:    page,
			size:    size,
			filters: filters,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userSelectQ)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery(regexp.QuoteMeta(userListQ)).
					WithArgs(size, (page-1)*size).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "email", "avatar",
						"is_active", "is_email_verified", "created_at", "updated_at",
					}))
			},
			expected: &dto.PaginatedUserResponse{
				Data:        []*md.User{},
				Count:       0,
				TotalPages:  0,
				CurrentPage: page,
				HasNextPage: false,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.ListUsers(context.Background(), tt.page, tt.size, tt.filters)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Count, result.Count)
				assert.Equal(t, tt.expected.TotalPages, result.TotalPages)
				assert.Equal(t, tt.expected.CurrentPage, result.CurrentPage)
				assert.Equal(t, tt.expected.HasNextPage, result.HasNextPage)

				if len(tt.expected.Data) > 0 {
					assert.Equal(t, len(tt.expected.Data), len(result.Data))
					for i, expectedUser := range tt.expected.Data {
						assert.Equal(t, expectedUser.ID, result.Data[i].ID)
						assert.Equal(t, expectedUser.Name, result.Data[i].Name)
					}
				} else {
					assert.Empty(t, result.Data)
				}
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	testErr := errors.New("test error")
	r := &Repository{conn: sqlxDB}

	testUser := &md.User{
		ID:              uuid.New(),
		Name:            "User 1",
		Email:           "user1@example.com",
		Avatar:          "avatar1.jpg",
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	tests := []struct {
		name        string
		mock        func()
		userID      uuid.UUID
		expected    *md.User
		expectedErr error
	}{
		{
			name:   "Success",
			userID: testUser.ID,
			mock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "avatar",
					"is_active", "is_email_verified", "created_at", "updated_at",
				})
				rows.AddRow(
					testUser.ID, testUser.Name, testUser.Email, testUser.Avatar,
					testUser.IsActive, testUser.IsEmailVerified, testUser.CreatedAt, testUser.UpdatedAt,
				)

				mock.ExpectQuery(regexp.QuoteMeta(userGetByIDQ)).
					WithArgs(testUser.ID).
					WillReturnRows(rows)
			},
			expected:    testUser,
			expectedErr: nil,
		},
		{
			name:   "ErrNotFound",
			userID: testUser.ID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userGetByIDQ)).
					WithArgs(testUser.ID).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    testUser,
			expectedErr: repo.ErrNotFound,
		},
		{
			name:   "ErrInternal",
			userID: testUser.ID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userGetByIDQ)).
					WithArgs(testUser.ID).
					WillReturnError(errors.New("test error"))
			},
			expected:    testUser,
			expectedErr: testErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := r.GetUserByID(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	testErr := errors.New("test error")
	r := &Repository{conn: sqlxDB}

	testUser := &md.User{
		ID:              uuid.New(),
		Name:            "User 1",
		Email:           "user1@example.com",
		Avatar:          "avatar1.jpg",
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	tests := []struct {
		name        string
		mock        func()
		expected    *md.User
		expectedErr error
	}{
		{
			name: "Success",
			mock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password", "avatar",
					"is_active", "is_email_verified", "created_at", "updated_at",
				})
				rows.AddRow(
					testUser.ID, testUser.Name, testUser.Email, testUser.Password, testUser.Avatar,
					testUser.IsActive, testUser.IsEmailVerified, testUser.CreatedAt, testUser.UpdatedAt,
				)

				mock.ExpectQuery(regexp.QuoteMeta(userGetByEmailQ)).
					WithArgs(testUser.Email).
					WillReturnRows(rows)
			},
			expected:    testUser,
			expectedErr: nil,
		},
		{
			name: "ErrNotFound",
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userGetByEmailQ)).
					WithArgs(testUser.Email).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    testUser,
			expectedErr: repo.ErrNotFound,
		},
		{
			name: "ErrInternal",
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(userGetByEmailQ)).
					WithArgs(testUser.Email).
					WillReturnError(errors.New("test error"))
			},
			expected:    testUser,
			expectedErr: testErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := r.GetUserByEmail(context.Background(), testUser.Email)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	testID := uuid.New()
	createReq := &dto.CreateUserRequest{
		Name:     "Test User",
		Password: "hashedpassword",
		Email:    "test@example.com",
		Avatar:   "avatar.jpg",
		IsActive: true,
		IsEmail:  true,
	}

	pgErr := &pgconn.PgError{Code: "23505"}

	tests := []struct {
		name        string
		req         *dto.CreateUserRequest
		mock        func()
		expectedID  uuid.UUID
		expectedErr error
	}{
		{
			name: "Success",
			req:  createReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(userCreateQ)).
					WithArgs(
						createReq.Name,
						createReq.Password,
						createReq.Email,
						createReq.Avatar,
						createReq.IsActive,
						createReq.IsEmail,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testID))
				mock.ExpectCommit()
			},
			expectedID:  testID,
			expectedErr: nil,
		},
		{
			name: "UserAlreadyExists",
			req:  createReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(userCreateQ)).
					WithArgs(
						createReq.Name,
						createReq.Password,
						createReq.Email,
						createReq.Avatar,
						createReq.IsActive,
						createReq.IsEmail,
					).
					WillReturnError(pgErr)
				mock.ExpectRollback()
			},
			expectedID:  uuid.Nil,
			expectedErr: repo.ErrAlreadyExists,
		},
		{
			name: "BeginTxError",
			req:  createReq,
			mock: func() {
				mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))
			},
			expectedID:  uuid.Nil,
			expectedErr: errors.New("tx begin error"),
		},
		{
			name: "CreateQueryError",
			req:  createReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(userCreateQ)).
					WithArgs(
						createReq.Name,
						createReq.Password,
						createReq.Email,
						createReq.Avatar,
						createReq.IsActive,
						createReq.IsEmail,
					).
					WillReturnError(errors.New("query error"))
				mock.ExpectRollback()
			},
			expectedID:  uuid.Nil,
			expectedErr: errors.New("query error"),
		},
		{
			name: "CommitError",
			req:  createReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(userCreateQ)).
					WithArgs(
						createReq.Name,
						createReq.Password,
						createReq.Email,
						createReq.Avatar,
						createReq.IsActive,
						createReq.IsEmail,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testID))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			expectedID:  uuid.Nil,
			expectedErr: errors.New("commit error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req != nil {
				tt.mock()
			}

			id, err := r.CreateUser(context.Background(), tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, repo.ErrAlreadyExists) {
					assert.ErrorIs(t, err, repo.ErrAlreadyExists)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
				assert.Equal(t, uuid.Nil, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	userID := uuid.New()
	updateReq := &dto.UpdateUserRequest{
		Name:     "Updated Name",
		Email:    "updated@example.com",
		Avatar:   "new-avatar.jpg",
		IsActive: true,
		IsEmail:  true,
	}

	tests := []struct {
		name        string
		id          uuid.UUID
		req         *dto.UpdateUserRequest
		mock        func()
		expectedErr error
	}{
		{
			name: "Success",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(userUpdateQ)).
					WithArgs(
						updateReq.Name,
						updateReq.Email,
						updateReq.Avatar,
						updateReq.IsActive,
						updateReq.IsEmail,
						userID,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name: "UserNotFound",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(userUpdateQ)).
					WithArgs(
						updateReq.Name,
						updateReq.Email,
						updateReq.Avatar,
						updateReq.IsActive,
						updateReq.IsEmail,
						userID,
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			expectedErr: repo.ErrNotFound,
		},
		{
			name: "BeginTxError",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))
			},
			expectedErr: errors.New("tx begin error"),
		},
		{
			name: "UpdateQueryError",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(userUpdateQ)).
					WithArgs(
						updateReq.Name,
						updateReq.Email,
						updateReq.Avatar,
						updateReq.IsActive,
						updateReq.IsEmail,
						userID,
					).
					WillReturnError(errors.New("update error"))
				mock.ExpectRollback()
			},
			expectedErr: errors.New("update error"),
		},
		{
			name: "RowsAffectedError",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(userUpdateQ)).
					WithArgs(
						updateReq.Name,
						updateReq.Email,
						updateReq.Avatar,
						updateReq.IsActive,
						updateReq.IsEmail,
						userID,
					).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
				mock.ExpectRollback()
			},
			expectedErr: errors.New("rows affected error"),
		},
		{
			name: "CommitError",
			id:   userID,
			req:  updateReq,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(userUpdateQ)).
					WithArgs(
						updateReq.Name,
						updateReq.Email,
						updateReq.Avatar,
						updateReq.IsActive,
						updateReq.IsEmail,
						userID,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			expectedErr: errors.New("commit error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req != nil && tt.id != uuid.Nil {
				tt.mock()
			}

			err := r.UpdateUser(context.Background(), tt.id, tt.req)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, repo.ErrNotFound) {
					assert.ErrorIs(t, err, repo.ErrNotFound)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	userID := uuid.New()
	tests := []struct {
		name        string
		id          uuid.UUID
		mock        func()
		expectedErr error
	}{
		{
			name: "Success",
			id:   userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(userDeleteQ)).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name: "UserNotFound",
			id:   userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(userDeleteQ)).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: repo.ErrNotFound,
		},
		{
			name: "DeleteError",
			id:   userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(userDeleteQ)).
					WithArgs(userID).
					WillReturnError(errors.New("delete error"))
			},
			expectedErr: errors.New("delete error"),
		},
		{
			name: "RowsAffectedError",
			id:   userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(userDeleteQ)).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedErr: errors.New("rows affected error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.id != uuid.Nil {
				tt.mock()
			}

			err := r.DeleteUser(context.Background(), tt.id)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, repo.ErrNotFound) {
					assert.ErrorIs(t, err, repo.ErrNotFound)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
