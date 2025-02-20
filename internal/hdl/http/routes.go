package http

import (
	"github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	metrics "github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	ot "github.com/opentracing/opentracing-go"
	"net/http"
	"time"
)

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/unprotected", h.endpoint)
	mux.HandleFunc("/api/protected", middleware.Apply(h.endpoint, middleware.Auth))
}

func (h *Handler) endpoint(w http.ResponseWriter, r *http.Request) {
	const op = "app.endpoint.hdl"
	s, c := time.Now(), http.StatusOK
	span, _ := ot.StartSpanFromContext(r.Context(), op)
	defer func() {
		span.Finish()
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	utils.StatusResponse(w, c)
}
