package ctrl

import (
	"context"
	"errors"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/cache"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestController_IsUserExist(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testEmail := "test@example.com"
	expectedResponse := &dto.ExistsUserResponse{Exists: true}
	emptyResponse := &dto.ExistsUserResponse{Exists: false}

	tests := []struct {
		name     string
		setup    func()
		email    string
		expected *dto.ExistsUserResponse
		wantErr  bool
		err      error
	}{
		{
			name: "UserExists",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(&md.User{}, nil)
			},
			email:    testEmail,
			expected: expectedResponse,
			wantErr:  false,
		},
		{
			name: "UserNotExists",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(nil, repo.ErrNotFound)
			},
			email:    testEmail,
			expected: emptyResponse,
			wantErr:  false,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(nil, errors.New("database error"))
			},
			email:    testEmail,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.IsUserExist(ctx, tt.email)

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

func TestController_ListUsers(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	page := 1
	size := 10
	filters := map[string]interface{}{"active": true}
	cacheKey := fmt.Sprintf(usersListKey, page, size, filters)

	successResponse := &dto.PaginatedUserResponse{
		Data: []*md.User{
			{ID: uuid.New(), Name: "User 1"},
			{ID: uuid.New(), Name: "User 2"},
		},
		CurrentPage: page,
	}

	tests := []struct {
		name     string
		setup    func()
		page     int
		size     int
		filters  map[string]interface{}
		expected *dto.PaginatedUserResponse
		wantErr  bool
	}{
		{
			name: "SuccessWithCacheHit",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
						return nil
					})
			},
			page:     page,
			size:     size,
			filters:  filters,
			expected: &dto.PaginatedUserResponse{},
			wantErr:  false,
		},
		{
			name: "SuccessWithCacheMiss",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					ListUsers(gomock.Any(), page, size, filters).
					Return(successResponse, nil)

				expectedBytes, _ := json.Marshal(successResponse)
				mockCache.EXPECT().
					Set(gomock.Any(), config.DefaultCacheTime, cacheKey, expectedBytes).
					Return()
			},
			page:     page,
			size:     size,
			filters:  filters,
			expected: successResponse,
			wantErr:  false,
		},
		{
			name: "CacheMissWithRepositoryError",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					ListUsers(gomock.Any(), page, size, filters).
					Return(nil, errors.New("database error"))
			},
			page:     page,
			size:     size,
			filters:  filters,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.ListUsers(ctx, tt.page, tt.size, tt.filters)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestController_GetUserByID(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	cacheKey := fmt.Sprintf(userCacheKey, testUserID)
	testUser := &md.User{
		ID:    testUserID,
		Name:  "Test User",
		Email: "test@example.com",
	}

	tests := []struct {
		name     string
		setup    func()
		userID   uuid.UUID
		expected *md.User
		wantErr  bool
		err      error
	}{
		{
			name: "SuccessWithCacheHit",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
						return nil
					})
			},
			userID:   testUserID,
			expected: &md.User{},
			wantErr:  false,
		},
		{
			name: "SuccessWithCacheMiss",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByID(gomock.Any(), testUserID).
					Return(testUser, nil)

				expectedBytes, _ := json.Marshal(testUser)
				mockCache.EXPECT().
					Set(gomock.Any(), config.DefaultCacheTime, cacheKey, expectedBytes).
					Return()
			},
			userID:   testUserID,
			expected: testUser,
			wantErr:  false,
		},
		{
			name: "UserNotFound",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByID(gomock.Any(), testUserID).
					Return(nil, repo.ErrNotFound)
			},
			userID:   testUserID,
			expected: nil,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByID(gomock.Any(), testUserID).
					Return(nil, errors.New("database error"))
			},
			userID:   testUserID,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.GetUserByID(ctx, tt.userID)

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

func TestController_GetUserByEmail(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testEmail := "test@example.com"
	cacheKey := fmt.Sprintf(userCacheKey, testEmail)
	testUser := &md.User{
		ID:    uuid.New(),
		Name:  "Test User",
		Email: testEmail,
	}

	tests := []struct {
		name     string
		setup    func()
		email    string
		expected *md.User
		wantErr  bool
		err      error
	}{
		{
			name: "SuccessWithCacheHit",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					DoAndReturn(func(ctx context.Context, key string, dest interface{}) error {
						return nil
					})
			},
			email:    testEmail,
			expected: &md.User{},
			wantErr:  false,
		},
		{
			name: "SuccessWithCacheMiss",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(testUser, nil)

				expectedBytes, _ := json.Marshal(testUser)
				mockCache.EXPECT().
					Set(gomock.Any(), config.DefaultCacheTime, cacheKey, expectedBytes).
					Return()
			},
			email:    testEmail,
			expected: testUser,
			wantErr:  false,
		},
		{
			name: "UserNotFound",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(nil, repo.ErrNotFound)
			},
			email:    testEmail,
			expected: nil,
			wantErr:  true,
			err:      ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockCache.EXPECT().
					GetToStruct(gomock.Any(), cacheKey, gomock.Any()).
					Return(cache.ErrNotFoundInCache)

				mockRepo.EXPECT().
					GetUserByEmail(gomock.Any(), testEmail).
					Return(nil, errors.New("database error"))
			},
			email:    testEmail,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.GetUserByEmail(ctx, tt.email)

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

func TestController_CreateUser(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	// Test data
	testUserID := uuid.New()
	testPassword := "securePassword123"
	testHash := "hashedPassword"
	testAvatarURL := "https://example.com/avatar.jpg"

	baseRequest := &dto.CreateUserRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: testPassword,
	}

	//withAvatarRequest := &dto.CreateUserRequest{
	//	Name:     "Test User",
	//	Email:    "test@example.com",
	//	Password: testPassword,
	//	Avatar:   testAvatarURL,
	//}

	testFile := &s3.UploadFileRequest{
		File:        []byte("testfile"),
		ContentType: "image/jpeg",
		Filename:    "avatar.jpg",
	}

	tests := []struct {
		name     string
		setup    func()
		request  *dto.CreateUserRequest
		file     *s3.UploadFileRequest
		expected *dto.CreateUserResponse
		wantErr  bool
		err      error
	}{
		{
			name: "SuccessWithoutAvatar",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), testPassword).
					Return(testHash, nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(testUserID, nil)

				mockCache.EXPECT().
					InvalidateKeysByPattern(gomock.Any(), userPattern).AnyTimes().Return()
			},
			request: baseRequest,
			file:    &s3.UploadFileRequest{},
			expected: &dto.CreateUserResponse{
				ID: testUserID,
			},
			wantErr: false,
		},
		{
			name: "SuccessWithAvatar",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), gomock.Any()).
					Return(testHash, nil)

				mockS3.EXPECT().
					UploadFile(gomock.Any(), testFile).
					Return(testAvatarURL, nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(testUserID, nil)

				mockCache.EXPECT().
					InvalidateKeysByPattern(gomock.Any(), userPattern).AnyTimes().Return()
			},
			request: baseRequest,
			file:    testFile,
			expected: &dto.CreateUserResponse{
				ID: testUserID,
			},
			wantErr: false,
		},
		{
			name: "PasswordHashError",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), gomock.Any()).
					Return("", errors.New("hashing error"))
			},
			request:  baseRequest,
			file:     nil,
			expected: nil,
			wantErr:  true,
		},
		{
			name: "S3UploadError",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), gomock.Any()).
					Return(testHash, nil)

				mockS3.EXPECT().
					UploadFile(gomock.Any(), testFile).
					Return("", errors.New("upload error"))
			},
			request:  baseRequest,
			file:     testFile,
			expected: nil,
			wantErr:  true,
		},
		{
			name: "UserAlreadyExists",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), gomock.Any()).
					Return(testHash, nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(uuid.Nil, repo.ErrAlreadyExists)
			},
			request:  baseRequest,
			file:     &s3.UploadFileRequest{},
			expected: nil,
			wantErr:  true,
			err:      ErrAlreadyExists,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockAuth.EXPECT().
					Hash(gomock.Any(), gomock.Any()).
					Return(testHash, nil)

				mockRepo.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(uuid.Nil, errors.New("database error"))
			},
			request:  baseRequest,
			file:     nil,
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result, err := ctrl.CreateUser(ctx, tt.request, tt.file)

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

func TestController_UpdateUser(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	testUserID := uuid.New()
	testAvatarURL := "https://example.com/avatar.jpg"
	testRequest := &dto.UpdateUserRequest{
		Name: "Updated Name",
	}
	testFile := &s3.UploadFileRequest{
		File:        []byte("testfile"),
		ContentType: "image/jpeg",
		Filename:    "avatar.png",
	}

	tests := []struct {
		name    string
		setup   func()
		id      uuid.UUID
		request *dto.UpdateUserRequest
		file    *s3.UploadFileRequest
		wantErr bool
		err     error
	}{
		{
			name: "SuccessWithoutAvatar",
			setup: func() {
				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), testUserID, testRequest).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), fmt.Sprintf(userCacheKey, testUserID)).
					Return()

				mockCache.EXPECT().
					InvalidateKeysByPattern(gomock.Any(), userPattern).
					Return().AnyTimes()
			},
			id:      testUserID,
			request: testRequest,
			file:    nil,
			wantErr: false,
		},
		{
			name: "SuccessWithAvatar",
			setup: func() {
				mockS3.EXPECT().
					UploadFile(gomock.Any(), testFile).
					Return(testAvatarURL, nil)

				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), testUserID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest) error {
						assert.Equal(t, testAvatarURL, req.Avatar)
						return nil
					})

				mockCache.EXPECT().
					Delete(gomock.Any(), fmt.Sprintf(userCacheKey, testUserID)).
					Return()

				mockCache.EXPECT().
					InvalidateKeysByPattern(gomock.Any(), userPattern).
					Return().AnyTimes()
			},
			id:      testUserID,
			request: testRequest,
			file:    testFile,
			wantErr: false,
		},
		{
			name: "UserNotFound",
			setup: func() {
				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), testUserID, testRequest).
					Return(repo.ErrNotFound)
			},
			id:      testUserID,
			request: testRequest,
			file:    nil,
			wantErr: true,
			err:     ErrNotFound,
		},
		{
			name: "S3UploadError",
			setup: func() {
				mockS3.EXPECT().
					UploadFile(gomock.Any(), testFile).
					Return("", errors.New("upload error"))
			},
			id:      testUserID,
			request: testRequest,
			file:    testFile,
			wantErr: true,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					UpdateUser(gomock.Any(), testUserID, testRequest).
					Return(errors.New("database error"))
			},
			id:      testUserID,
			request: testRequest,
			file:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := ctrl.UpdateUser(ctx, tt.id, tt.request, tt.file)

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

func TestController_DeleteUser(t *testing.T) {
	ctrlMock := gomock.NewController(t)
	defer ctrlMock.Finish()

	mockAuth := mocks.NewMockCore(ctrlMock)
	mockRepo := mocks.NewMockAppRepo(ctrlMock)
	mockCache := mocks.NewMockCacheService(ctrlMock)
	mockS3 := mocks.NewMockS3Service(ctrlMock)

	ctx := context.Background()
	ctrl := New(mockAuth, mockRepo, mockCache, mockS3, nil)

	// Test data
	testUserID := uuid.New()
	cacheKey := fmt.Sprintf(userCacheKey, testUserID)

	tests := []struct {
		name    string
		setup   func()
		userID  uuid.UUID
		wantErr bool
		err     error
	}{
		{
			name: "Success",
			setup: func() {
				mockRepo.EXPECT().
					DeleteUser(gomock.Any(), testUserID).
					Return(nil)

				mockCache.EXPECT().
					Delete(gomock.Any(), cacheKey).
					Return()

				mockCache.EXPECT().
					InvalidateKeysByPattern(gomock.Any(), userPattern).
					Return().AnyTimes()
			},
			userID:  testUserID,
			wantErr: false,
		},
		{
			name: "UserNotFound",
			setup: func() {
				mockRepo.EXPECT().
					DeleteUser(gomock.Any(), testUserID).
					Return(repo.ErrNotFound)
			},
			userID:  testUserID,
			wantErr: true,
			err:     ErrNotFound,
		},
		{
			name: "RepositoryError",
			setup: func() {
				mockRepo.EXPECT().
					DeleteUser(gomock.Any(), testUserID).
					Return(errors.New("database error"))
			},
			userID:  testUserID,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			err := ctrl.DeleteUser(ctx, tt.userID)

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
