package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/hdl"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/tests/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandler_ListDevices(t *testing.T) {
	const uri = "/device"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	testDevices := []models.Device{
		{
			ID:        uuid.New().String(),
			UserID:    testUUID,
			IP:        "192.168.1.1",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.New().String(),
			UserID:    testUUID,
			IP:        "192.168.1.2",
			CreatedAt: time.Now(),
		},
	}

	tests := []struct {
		name       string
		uid        any
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
				mctrl.EXPECT().ListDevices(gomock.Any(), testUUID).Return(nil, ctrl.ErrNotFound)
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
				mctrl.EXPECT().ListDevices(gomock.Any(), testUUID).Return(nil, testErr)
			},
		},
		{
			name:   "Success",
			uid:    testUUID,
			status: http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var devices []models.Device
				err := json.NewDecoder(r.Result().Body).Decode(&devices)
				assert.Nil(t, err)
				assert.Len(t, devices, 2)
				assert.Equal(t, testDevices[0].IP, devices[0].IP)
				assert.Equal(t, testDevices[1].IP, devices[1].IP)
			},
			expect: func() {
				mctrl.EXPECT().ListDevices(gomock.Any(), testUUID).Return(testDevices, nil)
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
			h.listDevices(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_GetDevice(t *testing.T) {
	const uriTemplate = "/device/%s"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	testDeviceID := uuid.New().String()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	testDevice := models.Device{
		ID:        testDeviceID,
		UserID:    testUUID,
		IP:        "192.168.1.1",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name       string
		deviceID   string
		uid        any
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:     "ErrToRetrievePathArg_Empty",
			deviceID: "",
			uid:      testUUID,
			status:   http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrToRetrievePathArg.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_Nil",
			deviceID: testDeviceID,
			uid:      uuid.Nil,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_InvalidType",
			deviceID: testDeviceID,
			uid:      "invalid-uuid",
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "StatusNotFound",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetDevice(gomock.Any(), testUUID, testDeviceID).Return(nil, ctrl.ErrNotFound)
			},
		},
		{
			name:     "StatusInternalServerError",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().GetDevice(gomock.Any(), testUUID, testDeviceID).Return(nil, testErr)
			},
		},
		{
			name:     "Success",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				var device models.Device
				err := json.NewDecoder(r.Result().Body).Decode(&device)
				assert.Nil(t, err)
				assert.Equal(t, testDevice.ID, device.ID)
				assert.Equal(t, testDevice.IP, device.IP)
			},
			expect: func() {
				mctrl.EXPECT().GetDevice(gomock.Any(), testUUID, testDeviceID).Return(&testDevice, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			uri := fmt.Sprintf(uriTemplate, tt.deviceID)
			req := httptest.NewRequest(http.MethodGet, uri, nil)

			ctx := context.WithValue(req.Context(), config.UidKey, tt.uid)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.deviceID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.getDevice(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_UpdateDevice(t *testing.T) {
	const uriTemplate = "/device/%s"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	testDeviceID := uuid.New().String()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	validRequest := map[string]interface{}{
		"name":      "Updated Device",
		"is_active": true,
	}

	invalidRequest := map[string]interface{}{
		"is_active": "not-a-boolean",
	}

	tests := []struct {
		name       string
		deviceID   string
		uid        any
		payload    any
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:     "ErrToRetrievePathArg_Empty",
			deviceID: "",
			uid:      testUUID,
			payload:  validRequest,
			status:   http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrToRetrievePathArg.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_Nil",
			deviceID: testDeviceID,
			uid:      uuid.Nil,
			payload:  validRequest,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_InvalidType",
			deviceID: testDeviceID,
			uid:      "invalid-uuid",
			payload:  validRequest,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrDecodeRequest",
			deviceID: testDeviceID,
			uid:      testUUID,
			payload:  invalidRequest,
			status:   http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Contains(t, res.Errors[0], "required rule")
			},
			expect: func() {},
		},
		{
			name:     "StatusNotFound",
			deviceID: testDeviceID,
			uid:      testUUID,
			payload:  validRequest,
			status:   http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().UpdateDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
					gomock.Any(),
				).Return(ctrl.ErrNotFound)
			},
		},
		{
			name:     "StatusInternalServerError",
			deviceID: testDeviceID,
			uid:      testUUID,
			payload:  validRequest,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().UpdateDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
					gomock.Any(),
				).Return(testErr)
			},
		},
		{
			name:     "Success",
			deviceID: testDeviceID,
			uid:      testUUID,
			payload:  validRequest,
			status:   http.StatusOK,
			assertions: func(r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, r.Result().StatusCode)
				assert.Equal(t, 0, r.Body.Len())
			},
			expect: func() {
				mctrl.EXPECT().UpdateDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
					gomock.Any(),
				).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			body, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			uri := fmt.Sprintf(uriTemplate, tt.deviceID)
			req := httptest.NewRequest(http.MethodPut, uri, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), config.UidKey, tt.uid)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.deviceID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.updateDevice(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}

func TestHandler_DeleteDevice(t *testing.T) {
	const uriTemplate = "/device/%s"
	mock := gomock.NewController(t)
	defer mock.Finish()

	testErr := errors.New("testErr")
	testUUID := uuid.New()
	testDeviceID := uuid.New().String()
	mctrl := mocks.NewMockAppCtrl(mock)
	mauth := mocks.NewMockCore(mock)
	h := New(mauth, mctrl)

	tests := []struct {
		name       string
		deviceID   string
		uid        any
		status     int
		expect     func()
		assertions func(r *httptest.ResponseRecorder)
	}{
		{
			name:     "ErrToRetrievePathArg_Empty",
			deviceID: "",
			uid:      testUUID,
			status:   http.StatusBadRequest,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrToRetrievePathArg.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_Nil",
			deviceID: testDeviceID,
			uid:      uuid.Nil,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "ErrFailedToParseUUID_InvalidType",
			deviceID: testDeviceID,
			uid:      "invalid-uuid",
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrFailedToParseUUID.Error(), res.Errors[0])
			},
			expect: func() {},
		},
		{
			name:     "StatusNotFound",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusNotFound,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, ctrl.ErrNotFound.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().DeleteDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
				).Return(ctrl.ErrNotFound)
			},
		},
		{
			name:     "StatusInternalServerError",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusInternalServerError,
			assertions: func(r *httptest.ResponseRecorder) {
				res := &utils.ErrorsResponse{}
				err := json.NewDecoder(r.Result().Body).Decode(res)
				assert.Nil(t, err)
				assert.Equal(t, hdl.ErrInternal.Error(), res.Errors[0])
			},
			expect: func() {
				mctrl.EXPECT().DeleteDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
				).Return(testErr)
			},
		},
		{
			name:     "Success_NoContent",
			deviceID: testDeviceID,
			uid:      testUUID,
			status:   http.StatusNoContent,
			assertions: func(r *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, r.Result().StatusCode)
				assert.Equal(t, 0, r.Body.Len())
			},
			expect: func() {
				mctrl.EXPECT().DeleteDevice(
					gomock.Any(),
					testUUID,
					testDeviceID,
				).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expect()

			uri := fmt.Sprintf(uriTemplate, tt.deviceID)
			req := httptest.NewRequest(http.MethodDelete, uri, nil)

			ctx := context.WithValue(req.Context(), config.UidKey, tt.uid)
			req = req.WithContext(ctx)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.deviceID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			h.deleteDevice(w, req)
			assert.Equal(t, tt.status, w.Result().StatusCode)

			defer func() {
				assert.Nil(t, w.Result().Body.Close())
			}()

			tt.assertions(w)
		})
	}
}
