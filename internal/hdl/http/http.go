package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "github.com/JMURv/golang-clean-template/api/rest/v1"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	mid "github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

type Handler struct {
	Router *chi.Mux
	au     auth.Core
	srv    *http.Server
	ctrl   ctrl.AppCtrl
}

func New(au auth.Core, ctrl ctrl.AppCtrl) *Handler {
	r := chi.NewRouter()
	r.Use(
		mid.Logger(zap.L()),
		middleware.StripSlashes,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		mid.Prometheus,
		mid.OT,
	)

	hdl := &Handler{
		Router: r,
		au:     au,
		ctrl:   ctrl,
	}

	hdl.RegisterAuthRoutes()
	hdl.RegisterUserRoutes()
	hdl.RegisterDeviceRoutes()
	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Get(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, "OK")
		},
	)
	return hdl
}

func (h *Handler) Start(port int) {
	h.srv = &http.Server{
		Handler:      h.Router,
		Addr:         fmt.Sprintf(":%v", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	zap.L().Info(
		"Starting HTTP server",
		zap.String("addr", h.srv.Addr),
	)

	err := h.srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		zap.L().Error("Server error", zap.Error(err))
	}
}

func (h *Handler) Close(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- h.srv.Shutdown(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
