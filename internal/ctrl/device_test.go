package ctrl

import (
	"context"
	"errors"
	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestController_ListDevices(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDevices := []md.Device{
		{
			ID:     uuid.New().String(),
			UserID: testUserID,
			IP:     "192.168.1.1",
			UA:     "test-user-agent-1",
		},
		{
			ID:     uuid.New().String(),
			UserID: testUserID,
			IP:     "192.168.1.2",
			UA:     "test-user-agent-2",
		},
	}

	tests := []struct {
		name     string
		setup    func()
		userID   uuid.UUID
		expected []md.Device
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					ListDevices(gomock.Any(), testUserID).
					Return(testDevices, nil)
			},
			userID:   testUserID,
			expected: testDevices,
			wantErr:  false,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					ListDevices(gomock.Any(), testUserID).
					Return(nil, errors.New("database error"))
			},
			userID:  testUserID,
			wantErr: true,
		},
		{
			name: "NoDevicesFound",
			setup: func() {
				mockRepo.EXPECT().
					ListDevices(gomock.Any(), testUserID).
					Return([]md.Device{}, nil)
			},
			userID:   testUserID,
			expected: []md.Device{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.ListDevices(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestController_GetDevice(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDeviceID := uuid.New().String()
	testDevice := &md.Device{
		ID:     testDeviceID,
		UserID: testUserID,
		IP:     "192.168.1.1",
		UA:     "test-user-agent",
	}

	tests := []struct {
		name     string
		setup    func()
		userID   uuid.UUID
		deviceID string
		expected *md.Device
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					GetDevice(gomock.Any(), testUserID, testDeviceID).
					Return(testDevice, nil)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			expected: testDevice,
			wantErr:  false,
		},
		{
			name: "DeviceNotFound",
			setup: func() {
				mockRepo.EXPECT().
					GetDevice(gomock.Any(), testUserID, testDeviceID).
					Return(nil, repo.ErrNotFound)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					GetDevice(gomock.Any(), testUserID, testDeviceID).
					Return(nil, errors.New("database error"))
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.GetDevice(ctx, tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestController_GetDeviceByID(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testDeviceID := uuid.New().String()
	testDevice := &md.Device{
		ID:     testDeviceID,
		UserID: uuid.New(),
		IP:     "192.168.1.1",
		UA:     "test-user-agent",
	}

	tests := []struct {
		name     string
		setup    func()
		deviceID string
		expected *md.Device
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					GetDeviceByID(gomock.Any(), testDeviceID).
					Return(testDevice, nil)
			},
			deviceID: testDeviceID,
			expected: testDevice,
			wantErr:  false,
		},
		{
			name: "DeviceNotFound",
			setup: func() {
				mockRepo.EXPECT().
					GetDeviceByID(gomock.Any(), testDeviceID).
					Return(nil, repo.ErrNotFound)
			},
			deviceID: testDeviceID,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					GetDeviceByID(gomock.Any(), testDeviceID).
					Return(nil, errors.New("database error"))
			},
			deviceID: testDeviceID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.GetDeviceByID(ctx, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestController_UpdateDevice(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDeviceID := uuid.New().String()
	validRequest := &dto.UpdateDeviceRequest{
		Name: "updated-device-name",
	}

	tests := []struct {
		name     string
		setup    func()
		userID   uuid.UUID
		deviceID string
		request  *dto.UpdateDeviceRequest
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					UpdateDevice(gomock.Any(), testUserID, testDeviceID, validRequest).
					Return(nil)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			request:  validRequest,
			wantErr:  false,
		},
		{
			name: "DeviceNotFound",
			setup: func() {
				mockRepo.EXPECT().
					UpdateDevice(gomock.Any(), testUserID, testDeviceID, validRequest).
					Return(repo.ErrNotFound)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			request:  validRequest,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					UpdateDevice(gomock.Any(), testUserID, testDeviceID, validRequest).
					Return(errors.New("database error"))
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			request:  validRequest,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := ctrl.UpdateDevice(ctx, tt.userID, tt.deviceID, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestController_DeleteDevice(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDeviceID := uuid.New().String()

	tests := []struct {
		name     string
		setup    func()
		userID   uuid.UUID
		deviceID string
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					DeleteDevice(gomock.Any(), testUserID, testDeviceID).
					Return(nil)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			wantErr:  false,
		},
		{
			name: "DeviceNotFound",
			setup: func() {
				mockRepo.EXPECT().
					DeleteDevice(gomock.Any(), testUserID, testDeviceID).
					Return(repo.ErrNotFound)
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					DeleteDevice(gomock.Any(), testUserID, testDeviceID).
					Return(errors.New("database error"))
			},
			userID:   testUserID,
			deviceID: testDeviceID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := ctrl.DeleteDevice(ctx, tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.ErrorIs(t, err, tt.err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
