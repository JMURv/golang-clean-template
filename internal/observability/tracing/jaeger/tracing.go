package jaeger

import (
	"context"
	"github.com/JMURv/golang-clean-template/internal/config"
	ot "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

func Start(ctx context.Context, serviceName string, conf *config.JaegerConfig) {
	cfg := jaeger.Configuration{
		ServiceName: serviceName,
		Sampler: &jaeger.SamplerConfig{
			Type:  conf.Sampler.Type,
			Param: conf.Sampler.Param,
		},
		Reporter: &jaeger.ReporterConfig{
			LogSpans:           conf.Reporter.LogSpans,
			LocalAgentHostPort: conf.Reporter.LocalAgentHostPort,
			CollectorEndpoint:  conf.Reporter.CollectorEndpoint,
		},
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		zap.L().Fatal("Error initializing Jaeger", zap.Error(err))
	}

	ot.SetGlobalTracer(tracer)
	zap.L().Info(
		"Jaeger has been started",
		zap.String("local agent", cfg.Reporter.LocalAgentHostPort),
		zap.String("collector", cfg.Reporter.CollectorEndpoint),
	)
	<-ctx.Done()

	if err = closer.Close(); err != nil {
		zap.L().Debug("Error shutting down Jaeger", zap.Error(err))
	}
	zap.L().Info("Jaeger has been stopped")
}
