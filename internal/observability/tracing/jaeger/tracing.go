package jaeger

import (
	"context"
	cfg "github.com/JMURv/golang-clean-template/internal/config"
	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

func Start(ctx context.Context, serviceName string, conf *cfg.JaegerConfig) {
	tracerCfg := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  conf.Sampler.Type,
			Param: float64(conf.Sampler.Param),
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           conf.Reporter.LogSpans,
			LocalAgentHostPort: conf.Reporter.LocalAgentHostPort,
		},
	}

	tracer, closer, err := tracerCfg.NewTracer()
	if err != nil {
		zap.L().Fatal("Error initializing Jaeger tracer", zap.Error(err))
	}

	opentracing.SetGlobalTracer(tracer)
	zap.L().Info("Jaeger has been started")
	<-ctx.Done()

	if err = closer.Close(); err != nil {
		zap.L().Debug("Error shutting down Jaeger", zap.Error(err))
	}
	zap.L().Info("Jaeger has been stopped")
}
