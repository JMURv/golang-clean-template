package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/JMURv/golang-clean-template/internal/hdl"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_ExistsUser(t *testing.T) {
	const uri = "/user/exists"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testEmail := "test@example.com"
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		payload    interface{}
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:    "ErrDecodeRequest_InvalidPayload",
			payload: "invalid-json",
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "decode request")
			},
			expect: func() {},
		},
		{
			name:    "ErrDecodeRequest_InvalidEmail",
			payload: map[string]interface{}{"email": ""},
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
			expect: func() {},
		},
		{
			name:    "StatusNotFound",
			payload: map[string]interface{}{"email": testEmail},
			status:  http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().IsUserExist(gomock.Any(), testEmail).Return(&dto.ExistsUserResponse{
					Exists: false,
				}, ctrl.ErrNotFound)
			},
		},
		{
			name:    "StatusInternalServerError",
			payload: map[string]interface{}{"email": testEmail},
			status:  http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, testErr.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().IsUserExist(gomock.Any(), testEmail).Return(&dto.ExistsUserResponse{
					Exists: false,
				}, testErr)
			},
		},
		{
			name:    "Success_UserExists",
			payload: map[string]any{"email": testEmail},
			status:  http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.ExistsUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, true, response.Exists)
			},
			expect: func() {
				mctrl.EXPECT().IsUserExist(gomock.Any(), testEmail).Return(&dto.ExistsUserResponse{
					Exists: true,
				}, nil)
			},
		},
		{
			name:    "Success_UserNotExists",
			payload: map[string]interface{}{"email": testEmail},
			status:  http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.ExistsUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, false, response.Exists)
			},
			expect: func() {
				mctrl.EXPECT().IsUserExist(gomock.Any(), testEmail).Return(&dto.ExistsUserResponse{
					Exists: false,
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			var body bytes.Buffer
			if strPayload, ok := tt.payload.(string); ok {
				body.WriteString(strPayload)
			} else {
				err := json.NewEncoder(&body).Encode(tt.payload)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, uri, &body)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			h.existsUser(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_ListUsers(t *testing.T) {
	const uri = "/users"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	testUsers := dto.PaginatedUserResponse{
		Data: []*md.User{
			{
				ID:       uuid.New(),
				Name:     "name",
				Email:    "example@email.com",
				IsActive: true,
			},
			{
				ID:       uuid.New(),
				Name:     "name-1",
				Email:    "example-1@email.com",
				IsActive: true,
			},
		},
		Count:       2,
		TotalPages:  1,
		CurrentPage: 1,
		HasNextPage: false,
	}

	tests := []struct {
		name       string
		query      string
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:   "DefaultPagination",
			query:  "",
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.PaginatedUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Len(t, response.Data, 2)
				assert.Equal(t, int64(2), response.Count)
				assert.Equal(t, config.DefaultPage, response.CurrentPage)
			},
			expect: func() {
				mctrl.EXPECT().ListUsers(
					gomock.Any(),
					config.DefaultPage,
					config.DefaultSize,
					gomock.Any(),
				).Return(&testUsers, nil)
			},
		},
		{
			name:   "CustomPagination",
			query:  "page=2&size=10",
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.PaginatedUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Len(t, response.Data, 2)
			},
			expect: func() {
				mctrl.EXPECT().ListUsers(
					gomock.Any(),
					2,
					10,
					gomock.Any(),
				).Return(&testUsers, nil)
			},
		},
		{
			name:   "InvalidPageParam",
			query:  "page=invalid&size=10",
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.PaginatedUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, 1, response.CurrentPage)
			},
			expect: func() {
				mctrl.EXPECT().ListUsers(
					gomock.Any(),
					1,
					10,
					gomock.Any(),
				).Return(&testUsers, nil)
			},
		},
		{
			name:   "InvalidSizeParam",
			query:  "page=2&size=invalid",
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.PaginatedUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, int64(2), response.Count)
			},
			expect: func() {
				mctrl.EXPECT().ListUsers(
					gomock.Any(),
					2,
					config.DefaultSize,
					gomock.Any(),
				).Return(&testUsers, nil)
			},
		},
		{
			name:   "StatusInternalServerError",
			query:  "",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().ListUsers(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, testErr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			req := httptest.NewRequest(http.MethodGet, uri+"?"+tt.query, nil)

			w := httptest.NewRecorder()
			h.listUsers(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_GetMe(t *testing.T) {
	const uri = "/users/me"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	testUser := md.User{
		ID:              testUUID,
		Name:            "Test User",
		Email:           "test@example.com",
		Avatar:          "https://example.com/avatar.jpg",
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	tests := []struct {
		name       string
		uid        interface{} // Can be uuid.UUID or other types
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:   "ErrFailedToParseUUID_Nil",
			uid:    uuid.Nil,
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "ErrFailedToParseUUID_InvalidType",
			uid:    "invalid-uuid",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "StatusNotFound",
			uid:    testUUID,
			status: http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(nil, ctrl.ErrNotFound)
			},
		},
		{
			name:   "StatusInternalServerError",
			uid:    testUUID,
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(nil, testErr)
			},
		},
		{
			name:   "Success",
			uid:    testUUID,
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var user md.User
				err := json.NewDecoder(r.Result().Body).Decode(&user)
				assert.Nil(t, err)
				assert.Equal(t, testUser.ID, user.ID)
				assert.Equal(t, testUser.Name, user.Name)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.Avatar, user.Avatar)
				assert.Equal(t, testUser.IsActive, user.IsActive)
				assert.Equal(t, testUser.IsEmailVerified, user.IsEmailVerified)
				assert.False(t, user.CreatedAt.IsZero())
				assert.False(t, user.UpdatedAt.IsZero())
				assert.Equal(t, "", user.Password)
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(&testUser, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			req := httptest.NewRequest(http.MethodGet, uri, nil)

			ctx := context.WithValue(req.Context(), config.UidKey, tt.uid)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			h.getMe(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_GetUser(t *testing.T) {
	const uriTemplate = "/users/%s"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	testUser := md.User{
		ID:              testUUID,
		Name:            "Test User",
		Email:           "test@example.com",
		Avatar:          "https://example.com/avatar.jpg",
		IsActive:        true,
		IsEmailVerified: true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	tests := []struct {
		name       string
		userID     string
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:   "ErrInvalidUUID_Empty",
			userID: "",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "ErrInvalidUUID_Format",
			userID: "invalid-uuid",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "StatusNotFound",
			userID: testUUID.String(),
			status: http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(nil, ctrl.ErrNotFound)
			},
		},
		{
			name:   "StatusInternalServerError",
			userID: testUUID.String(),
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(nil, testErr)
			},
		},
		{
			name:   "Success",
			userID: testUUID.String(),
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var user md.User
				err := json.NewDecoder(r.Result().Body).Decode(&user)
				assert.Nil(t, err)
				assert.Equal(t, testUser.ID, user.ID)
				assert.Equal(t, testUser.Name, user.Name)
				assert.Equal(t, testUser.Email, user.Email)
				assert.Equal(t, testUser.Avatar, user.Avatar)
				assert.Equal(t, testUser.IsActive, user.IsActive)
				assert.Equal(t, testUser.IsEmailVerified, user.IsEmailVerified)
				assert.False(t, user.CreatedAt.IsZero())
				assert.False(t, user.UpdatedAt.IsZero())
				assert.Empty(t, user.Password)
			},
			expect: func() {
				mctrl.EXPECT().GetUserByID(gomock.Any(), testUUID).Return(&testUser, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()
			uri := fmt.Sprintf(uriTemplate, tt.userID)
			req := httptest.NewRequest(http.MethodGet, uri, nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.getUser(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_CreateUser(t *testing.T) {
	const uri = "/users"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	validRequest := map[string]interface{}{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "securePassword123!",
	}

	invalidRequest := map[string]interface{}{
		"email": "invalid-email",
	}

	tests := []struct {
		name          string
		payload       interface{}
		withAvatar    bool
		avatarContent []byte
		status        int
		expect        func()
		assertions    func(r *httptest.ResponseRecorder)
	}{
		{
			name:    "ErrDecodeRequest_InvalidForm",
			payload: "invalid-form",
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrDecodeRequest.Error(), res.Errors[0])
			},
		},
		{
			name:    "ErrDecodeRequest_InvalidJSON",
			payload: "invalid-json",
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrDecodeRequest.Error(), res.Errors[0])
			},
		},
		{
			name:    "ErrValidation",
			payload: invalidRequest,
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
		},
		{
			name:          "ErrInvalidFileType",
			payload:       validRequest,
			withAvatar:    true,
			avatarContent: []byte("not an image"),
			status:        http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "invalid file type")
			},
		},
		{
			name:    "StatusConflict",
			payload: validRequest,
			status:  http.StatusConflict,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrAlreadyExists.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, ctrl.ErrAlreadyExists)
			},
		},
		{
			name:    "StatusInternalServerError",
			payload: validRequest,
			status:  http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil, testErr)
			},
		},
		{
			name:    "Success_WithoutAvatar",
			payload: validRequest,
			status:  http.StatusCreated,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.CreateUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, testUUID, response.ID)
			},
			expect: func() {
				mctrl.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dto.CreateUserResponse{ID: testUUID}, nil)
			},
		},
		{
			name:          "Success_WithValidImage",
			payload:       validRequest,
			withAvatar:    true,
			avatarContent: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			status:        http.StatusCreated,
			assertions: func(r *httptest.ResponseRecorder) {
				var response dto.CreateUserResponse
				err := json.NewDecoder(r.Result().Body).Decode(&response)
				assert.Nil(t, err)
				assert.Equal(t, testUUID, response.ID)
			},
			expect: func() {
				mctrl.EXPECT().CreateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&dto.CreateUserResponse{ID: testUUID}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expect != nil {
				tt.expect()
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			jsonData, err := json.Marshal(tt.payload)
			if err != nil {
				writer.WriteField("data", tt.payload.(string))
			} else {
				writer.WriteField("data", string(jsonData))
			}

			if tt.withAvatar && tt.avatarContent != nil {
				part, err := writer.CreateFormFile("avatar", "test.png")
				require.NoError(t, err)
				_, err = part.Write(tt.avatarContent)
				require.NoError(t, err)
			}

			require.NoError(t, writer.Close())

			req := httptest.NewRequest(http.MethodPost, uri, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()
			h.createUser(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			if tt.assertions != nil {
				tt.assertions(w)
			}
		})
	}
}

func TestHandler_UpdateUser(t *testing.T) {
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	validRequest := map[string]any{
		"name":  "Test User",
		"email": "test@example.com",
	}

	invalidRequest := map[string]any{
		"email": "invalid-email",
	}

	tests := []struct {
		name          string
		uid           string
		payload       any
		withAvatar    bool
		avatarContent []byte
		status        int
		expect        func()
		assertions    func(r *httptest.ResponseRecorder)
	}{
		{
			name:    "ErrFailedToParseUUID",
			uid:     "invalid-uuid",
			payload: validRequest,
			status:  http.StatusUnauthorized,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
		},
		{
			name:    "ErrDecodeRequest_InvalidForm",
			uid:     uuid.New().String(),
			payload: "invalid-form",
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrDecodeRequest.Error(), res.Errors[0])
			},
		},
		{
			name:    "ErrValidation",
			uid:     uuid.New().String(),
			payload: invalidRequest,
			status:  http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
		},
		{
			name:          "ErrInvalidFileType",
			uid:           uuid.New().String(),
			payload:       validRequest,
			withAvatar:    true,
			avatarContent: []byte("not an image"),
			status:        http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "invalid file type")
			},
		},
		{
			name:    "StatusInternalServerError",
			uid:     uuid.New().String(),
			payload: validRequest,
			status:  http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().UpdateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(testErr)
			},
		},
		{
			name:       "Success_WithoutAvatar",
			uid:        uuid.New().String(),
			payload:    validRequest,
			status:     http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {},
			expect: func() {
				mctrl.EXPECT().UpdateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil)
			},
		},
		{
			name:          "Success_WithValidImage",
			uid:           uuid.New().String(),
			payload:       validRequest,
			withAvatar:    true,
			avatarContent: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			status:        http.StatusOK,
			assertions:    func(r *httptest.ResponseRecorder) {},
			expect: func() {
				mctrl.EXPECT().UpdateUser(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expect != nil {
				tt.expect()
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			jsonData, err := json.Marshal(tt.payload)
			if err != nil {
				writer.WriteField("data", tt.payload.(string))
			} else {
				writer.WriteField("data", string(jsonData))
			}

			if tt.withAvatar && tt.avatarContent != nil {
				part, err := writer.CreateFormFile("avatar", "test.png")
				require.NoError(t, err)
				_, err = part.Write(tt.avatarContent)
				require.NoError(t, err)
			}

			require.NoError(t, writer.Close())

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.uid)

			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/users/%s", tt.uid), body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			ctx := context.WithValue(
				context.WithValue(
					req.Context(),
					chi.RouteCtxKey,
					rctx,
				),
				config.UidKey,
				tt.uid,
			)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			h.updateUser(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			if tt.assertions != nil {
				tt.assertions(w)
			}
		})
	}
}

func TestHandler_DeleteUser(t *testing.T) {
	const uriTemplate = "/users/%s"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		userID     string
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:   "ErrInvalidUUID_Empty",
			userID: "",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
		},
		{
			name:   "ErrInvalidUUID_Format",
			userID: "invalid-uuid",
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
		},
		{
			name:   "StatusNotFound",
			userID: testUUID.String(),
			status: http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().DeleteUser(
					gomock.Any(),
					testUUID,
				).Return(ctrl.ErrNotFound)
			},
		},
		{
			name:   "StatusInternalServerError",
			userID: testUUID.String(),
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().DeleteUser(
					gomock.Any(),
					testUUID,
				).Return(testErr)
			},
		},
		{
			name:   "Success",
			userID: testUUID.String(),
			status: http.StatusNoContent,
			assertions: func(r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, r.Result().StatusCode)
				assert.Equal(t, 0, r.Body.Len())
			},
			expect: func() {
				mctrl.EXPECT().DeleteUser(
					gomock.Any(),
					testUUID,
				).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expect != nil {
				tt.expect()
			}

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf(uriTemplate, tt.userID), nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.userID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.deleteUser(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			if tt.assertions != nil {
				tt.assertions(w)
			}
		})
	}
}
