package grpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/JMURv/golang-clean-template/api/grpc/v1/gen"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/hdl/grpc/interceptors"
	metrics "github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	pm "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	ot "github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

type Handler struct {
	gen.AppServer
	srv  *grpc.Server
	hsrv *health.Server
	ctrl ctrl.AppCtrl
}

func New(name string, ctrl ctrl.AppCtrl) *Handler {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.AuthUnaryInterceptor(),
			metrics.SrvMetrics.UnaryServerInterceptor(
				pm.WithExemplarFromContext(metrics.Exemplar),
			),
		),
		grpc.ChainStreamInterceptor(
			metrics.SrvMetrics.StreamServerInterceptor(
				pm.WithExemplarFromContext(metrics.Exemplar),
			),
		),
	)

	reflection.Register(srv)

	hsrv := health.NewServer()
	hsrv.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
	return &Handler{
		ctrl: ctrl,
		srv:  srv,
		hsrv: hsrv,
	}
}

func (h *Handler) Start(port int) {
	gen.RegisterAppServer(h.srv, h)
	grpc_health_v1.RegisterHealthServer(h.srv, h.hsrv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
	if err != nil {
		zap.L().Fatal("failed to listen", zap.Error(err))
	}

	if err = h.srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		zap.L().Fatal("failed to serve", zap.Error(err))
	}
}

func (h *Handler) Close() error {
	h.srv.GracefulStop()
	return nil
}

func (h *Handler) Procedure(ctx context.Context, req *gen.Empty) (*gen.Empty, error) {
	const op = "app.Procedure.hdl"
	s, c := time.Now(), codes.OK
	span, ctx := ot.StartSpanFromContext(ctx, op)
	defer func() {
		span.Finish()
		metrics.ObserveRequest(time.Since(s), int(c), op)
	}()

	return &gen.Empty{}, nil
}
