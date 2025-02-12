package http

import (
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	metrics "github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	"net/http"
	"time"
)

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/test", h.getTest)
}

func (h *Handler) getTest(w http.ResponseWriter, r *http.Request) {
	s, c := time.Now(), http.StatusOK
	const op = "app.getTest.hdl"
	defer func() {
		metrics.ObserveRequest(time.Since(s), c, op)
	}()

	utils.StatusResponse(w, c)
}
