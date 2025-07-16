package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestRepository_IsTokenValid(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()
	device := &md.Device{ID: "device123"}
	validToken := "valid-token"
	invalidToken := "invalid-token"

	tests := []struct {
		name        string
		userID      uuid.UUID
		device      *md.Device
		token       string
		mock        func()
		expected    bool
		expectedErr error
	}{
		{
			name:   "ValidToken",
			userID: userID,
			device: device,
			token:  validToken,
			mock: func() {
				rows := sqlmock.NewRows([]string{"token"}).AddRow(validToken)
				mock.ExpectQuery(regexp.QuoteMeta(isValidToken)).
					WithArgs(userID, device.ID).
					WillReturnRows(rows)
			},
			expected:    true,
			expectedErr: nil,
		},
		{
			name:   "InvalidToken",
			userID: userID,
			device: device,
			token:  invalidToken,
			mock: func() {
				rows := sqlmock.NewRows([]string{"token"}).AddRow(validToken)
				mock.ExpectQuery(regexp.QuoteMeta(isValidToken)).
					WithArgs(userID, device.ID).
					WillReturnRows(rows)
			},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:   "NoTokenFound",
			userID: userID,
			device: device,
			token:  validToken,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(isValidToken)).
					WithArgs(userID, device.ID).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    false,
			expectedErr: nil,
		},
		{
			name:   "DatabaseError",
			userID: userID,
			device: device,
			token:  validToken,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(isValidToken)).
					WithArgs(userID, device.ID).
					WillReturnError(errors.New("database error"))
			},
			expected:    false,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			valid, err := repo.IsTokenValid(context.Background(), tt.userID, tt.device, tt.token)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, valid)
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()
	hashedToken := "hashed-token"
	expiresAt := time.Now().Add(24 * time.Hour)
	device := &md.Device{
		ID:         "device123",
		Name:       "Test Device",
		DeviceType: "mobile",
		OS:         "iOS",
		Browser:    "Safari",
		UA:         "Mozilla/5.0",
		IP:         "192.168.1.1",
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		hashedT     string
		expiresAt   time.Time
		device      *md.Device
		mock        func()
		expectedErr error
	}{
		{
			name:      "Success",
			userID:    userID,
			hashedT:   hashedToken,
			expiresAt: expiresAt,
			device:    device,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(createUserDevice)).
					WithArgs(
						device.ID,
						userID,
						device.Name,
						device.DeviceType,
						device.OS,
						device.Browser,
						device.UA,
						device.IP,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(createRefreshToken)).
					WithArgs(
						userID,
						hashedToken,
						expiresAt,
						device.ID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedErr: nil,
		},
		{
			name:      "BeginTxError",
			userID:    userID,
			hashedT:   hashedToken,
			expiresAt: expiresAt,
			device:    device,
			mock: func() {
				mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))
			},
			expectedErr: errors.New("tx begin error"),
		},
		{
			name:      "CreateDeviceError",
			userID:    userID,
			hashedT:   hashedToken,
			expiresAt: expiresAt,
			device:    device,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(createUserDevice)).
					WithArgs(
						device.ID,
						userID,
						device.Name,
						device.DeviceType,
						device.OS,
						device.Browser,
						device.UA,
						device.IP,
					).
					WillReturnError(errors.New("device create error"))
				mock.ExpectRollback()
			},
			expectedErr: errors.New("device create error"),
		},
		{
			name:      "CreateTokenError",
			userID:    userID,
			hashedT:   hashedToken,
			expiresAt: expiresAt,
			device:    device,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(createUserDevice)).
					WithArgs(
						device.ID,
						userID,
						device.Name,
						device.DeviceType,
						device.OS,
						device.Browser,
						device.UA,
						device.IP,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(createRefreshToken)).
					WithArgs(
						userID,
						hashedToken,
						expiresAt,
						device.ID,
					).
					WillReturnError(errors.New("token create error"))
				mock.ExpectRollback()
			},
			expectedErr: errors.New("token create error"),
		},
		{
			name:      "CommitError",
			userID:    userID,
			hashedT:   hashedToken,
			expiresAt: expiresAt,
			device:    device,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(createUserDevice)).
					WithArgs(
						device.ID,
						userID,
						device.Name,
						device.DeviceType,
						device.OS,
						device.Browser,
						device.UA,
						device.IP,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(regexp.QuoteMeta(createRefreshToken)).
					WithArgs(
						userID,
						hashedToken,
						expiresAt,
						device.ID,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			expectedErr: errors.New("commit error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.CreateToken(context.Background(), tt.userID, tt.hashedT, tt.expiresAt, tt.device)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RevokeAllTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()

	tests := []struct {
		name        string
		userID      uuid.UUID
		mock        func()
		expectedErr error
	}{
		{
			name:   "Success",
			userID: userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeToken)).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:   "NoTokensToRevoke",
			userID: userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeToken)).
					WithArgs(userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: nil,
		},
		{
			name:   "DatabaseError",
			userID: userID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeToken)).
					WithArgs(userID).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.RevokeAllTokens(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()
	deviceID := "device123"
	testToken := &md.RefreshToken{
		UserID:    userID,
		TokenHash: "hashed-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		DeviceID:  deviceID,
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		deviceID    string
		mock        func()
		expected    *md.RefreshToken
		expectedErr error
	}{
		{
			name:     "Success",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				rows := sqlmock.NewRows([]string{"user_id", "token_hash", "expires_at", "device_id"}).
					AddRow(testToken.UserID, testToken.TokenHash, testToken.ExpiresAt, testToken.DeviceID)
				mock.ExpectQuery(regexp.QuoteMeta(getTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnRows(rows)
			},
			expected:    testToken,
			expectedErr: nil,
		},
		{
			name:     "NotFound",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: sql.ErrNoRows,
		},
		{
			name:     "DatabaseError",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.GetByDevice(context.Background(), tt.userID, tt.deviceID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, sql.ErrNoRows) {
					assert.ErrorIs(t, err, sql.ErrNoRows)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.expected != nil {
				assert.Equal(t, tt.expected.UserID, result.UserID)
				assert.Equal(t, tt.expected.DeviceID, result.DeviceID)
				assert.WithinDuration(t, tt.expected.ExpiresAt, result.ExpiresAt, time.Second)
			} else {
				assert.Nil(t, result)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_RevokeByDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()
	deviceID := "device123"

	tests := []struct {
		name        string
		userID      uuid.UUID
		deviceID    string
		mock        func()
		expectedErr error
	}{
		{
			name:     "Success",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:     "NoTokenFound",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: nil,
		},
		{
			name:     "DatabaseError",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.RevokeByDevice(context.Background(), tt.userID, tt.deviceID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
