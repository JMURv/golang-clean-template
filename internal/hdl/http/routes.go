package http

import (
	"github.com/JMURv/golang-clean-template/internal/hdl/http/middleware"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	metrics "github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	ot "github.com/opentracing/opentracing-go"
	"net/http"
	"time"
)

func (h *Handler) RegisterRoutes() {
	h.router.Get("/unprotected", h.endpoint)
	h.router.With(middleware.Auth(h.au)).Get("/protected", h.endpoint)
}

// endpoint godoc
//
//	@Summary		Test endpoint
//	@Description	Test endpoint description
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	nil
//	@Failure		500	{object}	utils.ErrorsResponse	"internal error"
//	@Router			/protected [get]
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
