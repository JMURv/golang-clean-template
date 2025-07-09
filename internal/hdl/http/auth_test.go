package http

import (
	"bytes"
	"context"
	"errors"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/auth/captcha"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/JMURv/golang-clean-template/internal/hdl"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Authenticate(t *testing.T) {
	const uri = "/auth/jwt"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		method     string
		passDevice bool
		status     int
		payload    map[string]any
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:       "ErrNoDeviceInfo",
			passDevice: true,
			method:     http.MethodPost,
			status:     http.StatusBadRequest,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ErrNoDeviceInfo.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "ErrDecodeRequest",
			method: http.MethodPost,
			status: http.StatusBadRequest,
			payload: map[string]any{
				"email":    0,
				"password": "password",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrDecodeRequest.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "ErrMissingEmail",
			method: http.MethodPost,
			status: http.StatusBadRequest,
			payload: map[string]any{
				"email":    "",
				"password": "password",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
			expect: func() {},
		},
		{
			name:   "ErrMissingPass",
			method: http.MethodPost,
			status: http.StatusBadRequest,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
			expect: func() {},
		},
		{
			name:   "VerifyRecaptcha failure",
			method: http.MethodPost,
			status: http.StatusInternalServerError,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(false, testErr)
			},
		},
		{
			name:   "ErrValidationFailed",
			method: http.MethodPost,
			status: http.StatusUnauthorized,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, captcha.ErrValidationFailed.Error(), res.Errors[0])
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(false, nil)
			},
		},
		{
			name:   "StatusNotFound",
			method: http.MethodPost,
			status: http.StatusNotFound,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(true, nil)
				mctrl.EXPECT().Authenticate(
					gomock.Any(), &dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					}, &dto.EmailAndPasswordRequest{
						Email:    "example@mail.com",
						Password: "password",
						Token:    "token",
					},
				).Return(nil, ctrl.ErrNotFound)
			},
		},
		{
			name:   "ErrInvalidCredentials",
			method: http.MethodPost,
			status: http.StatusUnauthorized,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, auth.ErrInvalidCredentials.Error(), res.Errors[0])
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(true, nil)
				mctrl.EXPECT().Authenticate(
					gomock.Any(), &dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					}, &dto.EmailAndPasswordRequest{
						Email:    "example@mail.com",
						Password: "password",
						Token:    "token",
					},
				).Return(nil, auth.ErrInvalidCredentials)
			},
		},
		{
			name:   "StatusInternalServerError",
			method: http.MethodPost,
			status: http.StatusInternalServerError,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, testErr.Error(), res.Errors[0])
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(true, nil)
				mctrl.EXPECT().Authenticate(
					gomock.Any(), &dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					}, &dto.EmailAndPasswordRequest{
						Email:    "example@mail.com",
						Password: "password",
						Token:    "token",
					},
				).Return(nil, testErr)
			},
		},
		{
			name:   "Success",
			method: http.MethodPost,
			status: http.StatusOK,
			payload: map[string]any{
				"email":    "example@mail.com",
				"password": "password",
				"token":    "token",
			},
			assertions: func(r *httptest.ResponseRecorder) {
				assert.Contains(t, r.Header().Get("Set-Cookie"), config.AccessCookieName)
			},
			expect: func() {
				mauth.EXPECT().VerifyRecaptcha(gomock.Any(), gomock.Any()).Return(true, nil)
				mctrl.EXPECT().Authenticate(
					gomock.Any(), &dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					}, &dto.EmailAndPasswordRequest{
						Email:    "example@mail.com",
						Password: "password",
						Token:    "token",
					},
				).Return(&dto.TokenPair{Access: "token", Refresh: "token"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				tt.expect()
				b, err := json.Marshal(tt.payload)
				require.NoError(t, err)

				req := httptest.NewRequest(tt.method, uri, bytes.NewBuffer(b))
				req.Header.Set("Content-Type", "application/json")
				if !tt.passDevice {
					ctx := context.WithValue(req.Context(), "ip", "0.0.0.0")
					ctx = context.WithValue(ctx, "ua", "user-agent")
					req = req.WithContext(ctx)
				}

				w := httptest.NewRecorder()
				h.authenticate(w, req)
				assert.Equal(t, tt.status, w.Result().StatusCode)

				defer func() {
					assert.Nil(t, w.Result().Body.Close())
				}()

				tt.assertions(w)
			},
		)
	}
}

func TestHandler_Refresh(t *testing.T) {
	const uri = "/auth/refresh"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		passDevice bool
		cookie     *http.Cookie
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:       "ErrNoDeviceInfo",
			passDevice: true,
			status:     http.StatusBadRequest,
			cookie:     &http.Cookie{Name: config.RefreshCookieName, Value: "refresh_token"},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ErrNoDeviceInfo.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "ErrMissingCookie",
			status: http.StatusBadRequest,
			cookie: nil,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrDecodeRequest.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:   "StatusNotFound",
			status: http.StatusNotFound,
			cookie: &http.Cookie{Name: config.RefreshCookieName, Value: "refresh_token"},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().Refresh(
					gomock.Any(),
					&dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					},
					&dto.RefreshRequest{
						Refresh: "refresh_token",
					},
				).Return(nil, ctrl.ErrNotFound)
			},
		},
		{
			name:   "ErrTokenRevoked",
			status: http.StatusUnauthorized,
			cookie: &http.Cookie{Name: config.RefreshCookieName, Value: "refresh_token"},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, auth.ErrTokenRevoked.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().Refresh(
					gomock.Any(),
					&dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					},
					&dto.RefreshRequest{
						Refresh: "refresh_token",
					},
				).Return(nil, auth.ErrTokenRevoked)
			},
		},
		{
			name:   "StatusInternalServerError",
			status: http.StatusInternalServerError,
			cookie: &http.Cookie{Name: config.RefreshCookieName, Value: "refresh_token"},
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().Refresh(
					gomock.Any(),
					&dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					},
					&dto.RefreshRequest{
						Refresh: "refresh_token",
					},
				).Return(nil, testErr)
			},
		},
		{
			name:   "Success",
			status: http.StatusOK,
			cookie: &http.Cookie{Name: config.RefreshCookieName, Value: "refresh_token"},
			assertions: func(r *httptest.ResponseRecorder) {
				assert.Contains(t, r.Header().Get("Set-Cookie"), config.AccessCookieName)
			},
			expect: func() {
				mctrl.EXPECT().Refresh(
					gomock.Any(),
					&dto.DeviceRequest{
						IP: "0.0.0.0",
						UA: "user-agent",
					},
					&dto.RefreshRequest{
						Refresh: "refresh_token",
					},
				).Return(&dto.TokenPair{
					Access:  "new_access",
					Refresh: "new_refresh",
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()
			req := httptest.NewRequest(http.MethodPost, uri, nil)

			if !tt.passDevice {
				ctx := context.WithValue(req.Context(), "ip", "0.0.0.0")
				ctx = context.WithValue(ctx, "ua", "user-agent")
				req = req.WithContext(ctx)
			}

			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			w := httptest.NewRecorder()
			h.refresh(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	const uri = "/auth/logout"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		uid        any
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:   "ErrFailedToGetUUID",
			uid:    "invalid-uuid", // Wrong type
			status: http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
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
				mctrl.EXPECT().Logout(gomock.Any(), testUUID).Return(ctrl.ErrNotFound)
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
				mctrl.EXPECT().Logout(gomock.Any(), testUUID).Return(testErr)
			},
		},
		{
			name:   "Success",
			uid:    testUUID,
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				cookies := r.Result().Cookies()
				assert.Len(t, cookies, 2)

				accessCookie := cookies[0]
				if accessCookie.Name != config.AccessCookieName {
					accessCookie = cookies[1]
				}
				assert.Equal(t, "", accessCookie.Value)
				assert.Equal(t, -1, accessCookie.MaxAge)

				refreshCookie := cookies[0]
				if refreshCookie.Name != config.RefreshCookieName {
					refreshCookie = cookies[1]
				}
				assert.Equal(t, "", refreshCookie.Value)
				assert.Equal(t, -1, refreshCookie.MaxAge)
			},
			expect: func() {
				mctrl.EXPECT().Logout(gomock.Any(), testUUID).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			req := httptest.NewRequest(http.MethodPost, uri, nil)

			ctx := context.WithValue(req.Context(), "uid", tt.uid)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			h.logout(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}
