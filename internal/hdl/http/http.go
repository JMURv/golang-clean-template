package http

import (
	"context"
	"fmt"
	_ "github.com/JMURv/golang-clean-template/api/rest/v1"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	mid "github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Handler struct {
	router *chi.Mux
	au     auth.Core
	srv    *http.Server
	ctrl   ctrl.AppCtrl
}

func New(au auth.Core, ctrl ctrl.AppCtrl) *Handler {
	r := chi.NewRouter()
	return &Handler{
		router: r,
		au:     au,
		ctrl:   ctrl,
	}
}

func (h *Handler) Start(port int) {
	h.router.Use(
		mid.Logger(zap.L()),
		middleware.StripSlashes,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		mid.Prometheus,
		mid.OT,
	)

	h.RegisterRoutes()
	h.router.Get("/swagger/*", httpSwagger.WrapHandler)
	h.router.Get(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, "OK")
		},
	)

	h.srv = &http.Server{
		Handler:      h.router,
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
	if err != nil && err != http.ErrServerClosed {
		zap.L().Error("Server error", zap.Error(err))
	}
}

func (h *Handler) Close(ctx context.Context) error {
	return h.srv.Shutdown(ctx)
}
