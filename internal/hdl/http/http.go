package http

import (
	"context"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	mid "github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Handler struct {
	srv  *http.Server
	ctrl ctrl.AppCtrl
}

func New(ctrl ctrl.AppCtrl) *Handler {
	return &Handler{
		ctrl: ctrl,
	}
}

func (h *Handler) Start(port int) {
	mux := http.NewServeMux()

	RegisterRoutes(mux, h)
	mux.HandleFunc(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, "OK")
		},
	)

	handler := mid.Logging(mux)
	handler = mid.RecoverPanic(handler)
	h.srv = &http.Server{
		Handler:      handler,
		Addr:         fmt.Sprintf(":%v", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	err := h.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		zap.L().Debug("Server error", zap.Error(err))
	}
}

func (h *Handler) Close(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
