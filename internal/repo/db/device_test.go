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
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
	"time"
)

func TestRepository_ListDevices(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := &Repository{conn: sqlxDB}

	userID := uuid.New()
	testDevices := []md.Device{
		{
			ID:         "device1",
			Name:       "Test Device 1",
			DeviceType: "mobile",
			OS:         "Android",
			Browser:    "Chrome",
			IP:         "192.168.1.1",
			UA:         "Mozilla/5.0",
			LastActive: time.Now(),
		},
		{
			ID:         "device2",
			Name:       "Test Device 2",
			DeviceType: "desktop",
			OS:         "Windows",
			Browser:    "Firefox",
			IP:         "192.168.1.2",
			UA:         "Mozilla/5.0",
			LastActive: time.Now(),
		},
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		mock        func()
		expected    []md.Device
		expectedErr error
	}{
		{
			name:   "SuccessWithDevices",
			userID: userID,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "device_type", "os", "user_agent", "browser", "ip", "last_active"}).
					AddRow(
						testDevices[0].ID,
						testDevices[0].Name,
						testDevices[0].DeviceType,
						testDevices[0].OS,
						testDevices[0].UA,
						testDevices[0].Browser,
						testDevices[0].IP,
						testDevices[0].LastActive,
					).
					AddRow(
						testDevices[1].ID,
						testDevices[1].Name,
						testDevices[1].DeviceType,
						testDevices[1].OS,
						testDevices[1].UA,
						testDevices[1].Browser,
						testDevices[1].IP,
						testDevices[1].LastActive,
					)
				mock.ExpectQuery(regexp.QuoteMeta(listDevices)).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			expected:    testDevices,
			expectedErr: nil,
		},
		{
			name:   "SuccessNoDevices",
			userID: userID,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "device_type", "os", "user_agent", "browser", "ip", "last_active"})
				mock.ExpectQuery(regexp.QuoteMeta(listDevices)).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			expected:    []md.Device{},
			expectedErr: nil,
		},
		{
			name:   "DatabaseError",
			userID: userID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(listDevices)).
					WithArgs(userID).
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			devices, err := repo.ListDevices(context.Background(), tt.userID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr.Error())
				assert.Nil(t, devices)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, devices)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	userID := uuid.New()
	deviceID := "device123"
	testDevice := md.Device{
		ID:         deviceID,
		Name:       "Test Device",
		DeviceType: "mobile",
		OS:         "Android",
		Browser:    "Chrome",
		IP:         "192.168.1.1",
		UA:         "Mozilla/5.0",
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		deviceID    string
		mock        func()
		expected    *md.Device
		expectedErr error
	}{
		{
			name:     "Success",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "device_type", "os", "user_agent", "browser", "ip", "last_active"}).
					AddRow(
						testDevice.ID,
						testDevice.Name,
						testDevice.DeviceType,
						testDevice.OS,
						testDevice.UA,
						testDevice.Browser,
						testDevice.IP,
						testDevice.LastActive,
					)
				mock.ExpectQuery(regexp.QuoteMeta(getDevice)).
					WithArgs(deviceID, userID).
					WillReturnRows(rows)
			},
			expected:    &testDevice,
			expectedErr: nil,
		},
		{
			name:     "NotFound",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getDevice)).
					WithArgs(deviceID, userID).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: repo.ErrNotFound,
		},
		{
			name:     "DatabaseError",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getDevice)).
					WithArgs(deviceID, userID).
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			device, err := r.GetDevice(context.Background(), tt.userID, tt.deviceID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, repo.ErrNotFound) {
					assert.ErrorIs(t, err, repo.ErrNotFound)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
				assert.Nil(t, device)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, device)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetDeviceByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	deviceID := "device123"
	testDevice := md.Device{
		ID:         deviceID,
		UserID:     uuid.New(),
		Name:       "Test Device",
		DeviceType: "mobile",
		OS:         "Android",
		Browser:    "Chrome",
		IP:         "192.168.1.1",
		UA:         "Mozilla/5.0",
	}

	tests := []struct {
		name        string
		deviceID    string
		mock        func()
		expected    *md.Device
		expectedErr error
	}{
		{
			name:     "Success",
			deviceID: deviceID,
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "user_id", "name", "device_type", "os", "browser", "user_agent", "ip", "last_active"}).
					AddRow(
						testDevice.ID,
						testDevice.UserID,
						testDevice.Name,
						testDevice.DeviceType,
						testDevice.OS,
						testDevice.Browser,
						testDevice.UA,
						testDevice.IP,
						testDevice.LastActive,
					)
				mock.ExpectQuery(regexp.QuoteMeta(getDeviceByID)).
					WithArgs(deviceID).
					WillReturnRows(rows)
			},
			expected:    &testDevice,
			expectedErr: nil,
		},
		{
			name:     "NotFound",
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getDeviceByID)).
					WithArgs(deviceID).
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: repo.ErrNotFound,
		},
		{
			name:     "DatabaseError",
			deviceID: deviceID,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(getDeviceByID)).
					WithArgs(deviceID).
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.deviceID != "" {
				tt.mock()
			}

			device, err := r.GetDeviceByID(context.Background(), tt.deviceID)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedErr, repo.ErrNotFound) {
					assert.ErrorIs(t, err, repo.ErrNotFound)
				} else {
					assert.EqualError(t, err, tt.expectedErr.Error())
				}
				assert.Nil(t, device)
			} else {
				assert.NoError(t, err)
				if assert.NotNil(t, device) {
					assert.Equal(t, tt.expected.ID, device.ID)
					assert.Equal(t, tt.expected.UserID, device.UserID)
					assert.Equal(t, tt.expected.Name, device.Name)
				}
			}
		})
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_UpdateDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

	userID := uuid.New()
	deviceID := "device123"
	updateReq := &dto.UpdateDeviceRequest{Name: "Updated Device Name"}

	tests := []struct {
		name        string
		userID      uuid.UUID
		deviceID    string
		req         *dto.UpdateDeviceRequest
		mock        func()
		expectedErr error
	}{
		{
			name:     "Success",
			userID:   userID,
			deviceID: deviceID,
			req:      updateReq,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(updateDevice)).
					WithArgs(updateReq.Name, deviceID, userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:     "NoRowsAffected",
			userID:   userID,
			deviceID: deviceID,
			req:      updateReq,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(updateDevice)).
					WithArgs(updateReq.Name, deviceID, userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: repo.ErrNotFound,
		},
		{
			name:     "DatabaseErrorOnExec",
			userID:   userID,
			deviceID: deviceID,
			req:      updateReq,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(updateDevice)).
					WithArgs(updateReq.Name, deviceID, userID).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		{
			name:     "DatabaseErrorOnRowsAffected",
			userID:   userID,
			deviceID: deviceID,
			req:      updateReq,
			mock: func() {
				result := sqlmock.NewErrorResult(errors.New("rows affected error"))
				mock.ExpectExec(regexp.QuoteMeta(updateDevice)).
					WithArgs(updateReq.Name, deviceID, userID).
					WillReturnResult(result)
			},
			expectedErr: errors.New("rows affected error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.deviceID != "" && tt.userID != uuid.Nil && tt.req != nil {
				tt.mock()
			}

			err := r.UpdateDevice(context.Background(), tt.userID, tt.deviceID, tt.req)

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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestRepository_DeleteDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	r := &Repository{conn: sqlxDB}

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
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(regexp.QuoteMeta(deleteDevice)).
					WithArgs(deviceID, userID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedErr: nil,
		},
		{
			name:     "NoRowsAffected",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(regexp.QuoteMeta(deleteDevice)).
					WithArgs(deviceID, userID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: repo.ErrNotFound,
		},
		{
			name:     "DatabaseErrorOnExec",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec(regexp.QuoteMeta(deleteDevice)).
					WithArgs(deviceID, userID).
					WillReturnError(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		{
			name:     "DatabaseErrorOnRowsAffected",
			userID:   userID,
			deviceID: deviceID,
			mock: func() {
				mock.ExpectExec(regexp.QuoteMeta(revokeTokenByDevice)).
					WithArgs(userID, deviceID).
					WillReturnResult(sqlmock.NewResult(1, 1))

				result := sqlmock.NewErrorResult(errors.New("rows affected error"))
				mock.ExpectExec(regexp.QuoteMeta(deleteDevice)).
					WithArgs(deviceID, userID).
					WillReturnResult(result)
			},
			expectedErr: errors.New("rows affected error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.deviceID != "" && tt.userID != uuid.Nil {
				tt.mock()
			}

			err := r.DeleteDevice(context.Background(), tt.userID, tt.deviceID)

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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
