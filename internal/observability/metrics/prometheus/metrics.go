package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var SrvMetrics = grpcprom.NewServerMetrics(
	grpcprom.WithServerHandlingTimeHistogram(
		grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
	),
)

var Exemplar = func(ctx context.Context) prometheus.Labels {
	return prometheus.Labels{"traceID": strconv.Itoa(1)}
}

type Metric struct {
	srv *http.Server
	reg *prometheus.Registry
}

func New(port int) *Metric {
	return &Metric{
		srv: &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		},
		reg: prometheus.NewRegistry(),
	}
}

func (m *Metric) Start(ctx context.Context) {
	m.reg.MustRegister(
		SrvMetrics,
		RequestMetrics,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	mux := http.NewServeMux()
	mux.Handle(
		"/metrics", promhttp.HandlerFor(
			m.reg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		),
	)

	m.srv.Handler = mux
	go func() {
		if err := m.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("Prometheus server has been stopped with error", zap.Error(err))
		}
	}()
	zap.L().Info("Prometheus server has been started", zap.String("addr", m.srv.Addr))

	<-ctx.Done()
	if err := m.srv.Shutdown(ctx); err != nil {
		zap.L().Error("Prometheus server shutdown failed", zap.Error(err))
	}
	zap.L().Info("Prometheus server has been stopped")
}

var RequestMetrics = promauto.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace:  "svc",
		Name:       "request_metrics",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"status", "endpoint"},
)

func ObserveRequest(d time.Duration, status int, endpoint string) {
	RequestMetrics.WithLabelValues(strconv.Itoa(status), endpoint).Observe(d.Seconds())
}
