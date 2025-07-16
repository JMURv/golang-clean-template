package ctrl

import (
	"context"
	"errors"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/auth/jwt"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func TestController_Authenticate(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDevice := &dto.DeviceRequest{
		IP: "192.168.1.1",
		UA: "test-user-agent",
	}

	testRequest := &dto.EmailAndPasswordRequest{
		Email:    "test@example.com",
		Password: "validpassword123!",
	}
	testTokenPair := &dto.TokenPair{
		Access:  "access-token",
		Refresh: "refresh-token",
	}

	testUser := &models.User{
		ID:       testUserID,
		Email:    "test@example.com",
		Password: "$2a$10$hashedpassword",
	}

	tests := []struct {
		name     string
		setup    func()
		input    *dto.EmailAndPasswordRequest
		expected *dto.TokenPair
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testRequest.Email).
					Return(testUser, nil)
				mockAuth.EXPECT().
					ComparePasswords([]byte(testUser.Password), []byte(testRequest.Password)).
					Return(nil)
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return(testTokenPair.Access, testTokenPair.Refresh, nil)
				mockAuth.EXPECT().
					GetRefreshTime().
					Return(time.Now())
				mockRepo.EXPECT().
					CreateToken(gomock.Any(), testUserID, testTokenPair.Refresh, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			input:    testRequest,
			expected: testTokenPair,
			wantErr:  false,
		},
		{
			name: "UserNotFound",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testRequest.Email).
					Return(nil, repo.ErrNotFound)
			},
			input:   testRequest,
			wantErr: true,
			err:     ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testRequest.Email).
					Return(nil, errors.New("db error"))
			},
			input:   testRequest,
			wantErr: true,
		},
		{
			name: "InvalidCredentials",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testRequest.Email).
					Return(testUser, nil)
				mockAuth.EXPECT().
					ComparePasswords([]byte(testUser.Password), []byte(testRequest.Password)).
					Return(auth.ErrInvalidCredentials)
			},
			input:   testRequest,
			wantErr: true,
			err:     auth.ErrInvalidCredentials,
		},
		{
			name: "TokenGenerationError",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testRequest.Email).
					Return(testUser, nil)
				mockAuth.EXPECT().
					ComparePasswords([]byte(testUser.Password), []byte(testRequest.Password)).
					Return(nil)
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return("", "", errors.New("token error"))
			},
			input:   testRequest,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.Authenticate(ctx, testDevice, tt.input)

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

func TestController_Refresh(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testDevice := &dto.DeviceRequest{
		IP: "192.168.1.1",
		UA: "test-user-agent",
	}
	testRefreshToken := "valid-refresh-token"
	testRequest := &dto.RefreshRequest{
		Refresh: testRefreshToken,
	}
	testTokenPair := &dto.TokenPair{
		Access:  "new-access-token",
		Refresh: "new-refresh-token",
	}

	testClaims := jwt.Claims{
		UID: testUserID,
	}

	tests := []struct {
		name     string
		setup    func()
		input    *dto.RefreshRequest
		expected *dto.TokenPair
		wantErr  bool
		err      error
	}{
		{
			name: "Success",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(true, nil)
				mockAuth.EXPECT().
					GetRefreshTime().
					Return(time.Now())
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return(testTokenPair.Access, testTokenPair.Refresh, nil)
				mockRepo.EXPECT().
					RevokeAllTokens(gomock.Any(), testUserID).
					Return(nil)
				mockRepo.EXPECT().
					CreateToken(gomock.Any(), testUserID, testTokenPair.Refresh, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			input:    testRequest,
			expected: testTokenPair,
			wantErr:  false,
		},
		{
			name: "InvalidToken",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(jwt.Claims{}, auth.ErrInvalidToken)
			},
			input:   testRequest,
			wantErr: true,
			err:     auth.ErrInvalidToken,
		},
		{
			name: "TokenRevoked",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(false, nil)
			},
			input:   testRequest,
			wantErr: true,
			err:     auth.ErrTokenRevoked,
		},
		{
			name: "TokenValidationError",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(false, errors.New("db error"))
			},
			input:   testRequest,
			wantErr: true,
		},
		{
			name: "TokenGenerationError",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(true, nil)
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return("", "", errors.New("token error"))
			},
			input:   testRequest,
			wantErr: true,
		},
		{
			name: "RevokeTokensError",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(true, nil)
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return(testTokenPair.Access, testTokenPair.Refresh, nil)
				mockRepo.EXPECT().
					RevokeAllTokens(gomock.Any(), testUserID).
					Return(errors.New("revoke error"))
			},
			input:   testRequest,
			wantErr: true,
		},
		{
			name: "CreateTokenError",
			setup: func() {
				mockAuth.EXPECT().
					ParseClaims(gomock.Any(), testRefreshToken).
					Return(testClaims, nil)
				mockRepo.EXPECT().
					IsTokenValid(gomock.Any(), testUserID, gomock.Any(), testRefreshToken).
					Return(true, nil)
				mockAuth.EXPECT().
					GetRefreshTime().
					Return(time.Now())
				mockAuth.EXPECT().
					GenPair(gomock.Any(), testUserID).
					Return(testTokenPair.Access, testTokenPair.Refresh, nil)
				mockRepo.EXPECT().
					RevokeAllTokens(gomock.Any(), testUserID).
					Return(nil)
				mockRepo.EXPECT().
					CreateToken(gomock.Any(), testUserID, testTokenPair.Refresh, gomock.Any(), gomock.Any()).
					Return(errors.New("create error"))
			},
			input:   testRequest,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.Refresh(ctx, testDevice, tt.input)

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

func TestController_Logout(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()

	tests := []struct {
		name    string
		setup   func()
		input   uuid.UUID
		wantErr bool
		err     error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					RevokeAllTokens(gomock.Any(), testUserID).
					Return(nil)
			},
			input:   testUserID,
			wantErr: false,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					RevokeAllTokens(gomock.Any(), testUserID).
					Return(errors.New("database error"))
			},
			input:   testUserID,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := ctrl.Logout(ctx, tt.input)

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
