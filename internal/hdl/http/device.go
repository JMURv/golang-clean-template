package http

import (
	"errors"
	"net/http"

	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/JMURv/golang-clean-template/internal/hdl"
	mid "github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	_ "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (h *Handler) RegisterDeviceRoutes() {
	h.router.With(mid.Auth(h.au, mid.AuthOpts{})).Get("/device", h.listDevices)
	h.router.With(mid.Auth(h.au, mid.AuthOpts{})).Get("/device/{id}", h.getDevice)
	h.router.With(mid.Auth(h.au, mid.AuthOpts{})).Put("/device/{id}", h.updateDevice)
	h.router.With(mid.Auth(h.au, mid.AuthOpts{})).Delete("/device/{id}", h.deleteDevice)
}

// listDevices godoc
//
//	@Summary		List all devices for the authenticated user
//	@Description	Retrieve a list of registered devices for the current user
//	@Tags			Device
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authorization token"
//	@Success		200				{array}		[]models.Device
//	@Failure		404				{object}	utils.ErrorsResponse	"no devices found"
//	@Failure		500				{object}	utils.ErrorsResponse	"internal error"
//	@Router			/device [get]
func (h *Handler) listDevices(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value("uid").(uuid.UUID)
	if uid == uuid.Nil || !ok {
		zap.L().Error(
			hdl.ErrFailedToParseUUID.Error(),
			zap.Any("uid", r.Context().Value("uid")),
		)
		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrFailedToParseUUID)
		return
	}

	res, err := h.ctrl.ListDevices(r.Context(), uid)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	utils.SuccessResponse(w, http.StatusOK, res)
}

// getDevice godoc
//
//	@Summary		Get a specific device by ID
//	@Description	Retrieve details of a device owned by the current user
//	@Tags			Device
//	@Param			id	path	string	true	"Device UUID"
//	@Produce		json
//	@Param			Authorization	header		string	true	"Authorization token"
//	@Success		200				{object}	models.Device
//	@Failure		400				{object}	utils.ErrorsResponse	"invalid device ID path parameter"
//	@Failure		404				{object}	utils.ErrorsResponse	"device not found"
//	@Failure		500				{object}	utils.ErrorsResponse	"internal error"
//	@Router			/device/{id} [get]
func (h *Handler) getDevice(w http.ResponseWriter, r *http.Request) {
	dID := chi.URLParam(r, "id")
	if dID == "" {
		zap.L().Error(
			hdl.ErrToRetrievePathArg.Error(),
			zap.String("path", r.URL.Path),
		)
		utils.ErrResponse(w, http.StatusBadRequest, hdl.ErrToRetrievePathArg)
		return
	}

	uid, ok := r.Context().Value("uid").(uuid.UUID)
	if uid == uuid.Nil || !ok {
		zap.L().Error(
			hdl.ErrFailedToParseUUID.Error(),
			zap.Any("uid", r.Context().Value("uid")),
		)
		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrFailedToParseUUID)
		return
	}

	res, err := h.ctrl.GetDevice(r.Context(), uid, dID)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	utils.SuccessResponse(w, http.StatusOK, res)
}

// updateDevice godoc
//
//	@Summary		Update a device
//	@Description	Modify properties of a device owned by the current user
//	@Tags			Device
//	@Param			id		path	string					true	"Device UUID"
//	@Param			body	body	dto.UpdateDeviceRequest	true	"Update payload"
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header	string	true	"Authorization token"
//	@Success		200				"OK"
//	@Failure		400				{object}	utils.ErrorsResponse	"invalid device ID or payload"
//	@Failure		404				{object}	utils.ErrorsResponse	"device not found"
//	@Failure		500				{object}	utils.ErrorsResponse	"internal error"
//	@Router			/device/{id} [put]
func (h *Handler) updateDevice(w http.ResponseWriter, r *http.Request) {
	dID := chi.URLParam(r, "id")
	if dID == "" {
		zap.L().Debug(
			hdl.ErrToRetrievePathArg.Error(),
			zap.String("path", r.URL.Path),
		)
		utils.ErrResponse(w, http.StatusBadRequest, hdl.ErrToRetrievePathArg)
		return
	}

	uid, ok := r.Context().Value("uid").(uuid.UUID)
	if uid == uuid.Nil || !ok {
		zap.L().Error(
			hdl.ErrFailedToParseUUID.Error(),
			zap.Any("uid", r.Context().Value("uid")),
		)
		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrFailedToParseUUID)
		return
	}

	req := &dto.UpdateDeviceRequest{}
	if ok = utils.ParseAndValidate(w, r, req); !ok {
		return
	}

	err := h.ctrl.UpdateDevice(r.Context(), uid, dID, req)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	utils.StatusResponse(w, http.StatusOK)
}

// deleteDevice godoc
//
//	@Summary		Delete a device
//	@Description	Remove a device owned by the current user
//	@Tags			Device
//	@Param			id	path	string	true	"Device UUID"
//	@Produce		json
//	@Param			Authorization	header	string	true	"Authorization token"
//	@Success		204				"No Content"
//	@Failure		400				{object}	utils.ErrorsResponse	"invalid device ID"
//	@Failure		404				{object}	utils.ErrorsResponse	"device not found"
//	@Failure		500				{object}	utils.ErrorsResponse	"internal error"
//	@Router			/device/{id} [delete]
func (h *Handler) deleteDevice(w http.ResponseWriter, r *http.Request) {
	dID := chi.URLParam(r, "id")
	if dID == "" {
		zap.L().Error(
			hdl.ErrToRetrievePathArg.Error(),
			zap.String("path", r.URL.Path),
		)
		utils.ErrResponse(w, http.StatusBadRequest, hdl.ErrToRetrievePathArg)
		return
	}

	uid, ok := r.Context().Value("uid").(uuid.UUID)
	if uid == uuid.Nil || !ok {
		zap.L().Error(
			hdl.ErrFailedToParseUUID.Error(),
			zap.Any("uid", r.Context().Value("uid")),
		)
		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrFailedToParseUUID)
		return
	}

	err := h.ctrl.DeleteDevice(r.Context(), uid, dID)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	utils.StatusResponse(w, http.StatusNoContent)
}
