package http

import (
	"errors"
	"net/http"

	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/auth/captcha"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/JMURv/golang-clean-template/internal/hdl"
	mid "github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (h *Handler) RegisterAuthRoutes() {
	h.router.With(mid.Device).Post("/auth/jwt", h.authenticate)
	h.router.With(mid.Device).Post("/auth/jwt/refresh", h.refresh)
	h.router.With(mid.Auth(h.au, mid.AuthOpts{})).Post("/auth/logout", h.logout)
}

// authenticate godoc
//
//	@Summary		Authenticate using email & password
//	@Description	Verify reCAPTCHA, then authenticate and set JWT cookies
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			X-Real-IP	header						string	true	"Client real IP address"
//	@Param			User-Agent	header						string	true	"Client User-Agent"
//	@Param			body		dto.EmailAndPasswordRequest	true	"email, password, reCAPTCHA token"
//	@Success		200			{object}					dto.TokenPair
//	@Failure		400			{object}					utils.ErrorsResponse	"missing device info or bad payload"
//	@Failure		401			{object}					utils.ErrorsResponse	"invalid credentials or reCAPTCHA"
//	@Failure		404			{object}					utils.ErrorsResponse	"user not found"
//	@Failure		500			{object}					utils.ErrorsResponse	"internal error"
//	@Router			/auth/jwt [post]
func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) {
	d, ok := utils.ParseDeviceByRequest(r)
	if !ok {
		utils.ErrResponse(w, http.StatusBadRequest, ErrNoDeviceInfo)
		return
	}

	req := &dto.EmailAndPasswordRequest{}
	if ok = utils.ParseAndValidate(w, r, req); !ok {
		return
	}

	valid, err := h.au.VerifyRecaptcha(req.Token, captcha.PassAuth)
	if err != nil {
		utils.ErrResponse(w, http.StatusInternalServerError, captcha.ErrVerificationFailed)
		return
	}

	if !valid {
		utils.ErrResponse(w, http.StatusUnauthorized, captcha.ErrValidationFailed)
		return
	}

	res, err := h.ctrl.Authenticate(r.Context(), &d, req)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		}

		if errors.Is(err, auth.ErrInvalidCredentials) {
			utils.ErrResponse(w, http.StatusUnauthorized, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	utils.SetAuthCookies(w, res.Access, res.Refresh)
	utils.StatusResponse(w, http.StatusOK)
}

// refresh godoc
//
//	@Summary		Refresh JWT tokens
//	@Description	Validate device header and refresh tokens, reset cookies
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			X-Real-IP	header				string	true	"Client real IP address"
//	@Param			User-Agent	header				string	true	"Client User-Agent"
//	@Param			body		dto.RefreshRequest	true	"refresh_token"
//	@Success		200			{object}			dto.TokenPair
//	@Failure		400			{object}			utils.ErrorsResponse	"missing device info or bad payload"
//	@Failure		401			{object}			utils.ErrorsResponse	"token revoked or invalid"
//	@Failure		404			{object}			utils.ErrorsResponse	"session not found"
//	@Failure		500			{object}			utils.ErrorsResponse	"internal error"
//	@Router			/auth/jwt/refresh [post]
func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	d, ok := utils.ParseDeviceByRequest(r)
	if !ok {
		utils.ErrResponse(w, http.StatusBadRequest, ErrNoDeviceInfo)
		return
	}

	cookie, err := r.Cookie(config.RefreshCookieName)
	if err != nil {
		utils.ErrResponse(w, http.StatusBadRequest, hdl.ErrDecodeRequest)
		return
	}

	res, err := h.ctrl.Refresh(
		r.Context(), &d, &dto.RefreshRequest{
			Refresh: cookie.Value,
		},
	)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		} else if errors.Is(err, auth.ErrTokenRevoked) {
			utils.ErrResponse(w, http.StatusUnauthorized, err)
			return
		}

		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	utils.SetAuthCookies(w, res.Access, res.Refresh)
	utils.StatusResponse(w, http.StatusOK)
}

// logout godoc
//
//	@Summary		Logout user
//	@Description	Revoke refresh token, clear JWT cookies
//	@Tags			Authentication
//	@Produce		json
//	@Param			Authorization	header		string					true	"Authorization token"
//	@Success		200				{object}	nil						"OK"
//	@Failure		404				{object}	utils.ErrorsResponse	"session not found"
//	@Failure		500				{object}	utils.ErrorsResponse	"internal error"
//	@Router			/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	uid, ok := r.Context().Value("uid").(uuid.UUID)
	if !ok {
		zap.L().Error(
			hdl.ErrFailedToGetUUID.Error(),
			zap.Any("uid", r.Context().Value("uid")),
		)
		utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
		return
	}

	err := h.ctrl.Logout(r.Context(), uid)
	if err != nil {
		if errors.Is(err, ctrl.ErrNotFound) {
			utils.ErrResponse(w, http.StatusNotFound, err)
			return
		} else {
			utils.ErrResponse(w, http.StatusInternalServerError, hdl.ErrInternal)
			return
		}
	}

	http.SetCookie(
		w, &http.Cookie{
			Name:     config.AccessCookieName,
			Value:    "",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		},
	)

	http.SetCookie(
		w, &http.Cookie{
			Name:     config.RefreshCookieName,
			Value:    "",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
		},
	)

	utils.StatusResponse(w, http.StatusOK)
}
