package jaeger

import (
	"context"
	"github.com/JMURv/golang-clean-template/internal/config"
	ot "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
)

func Start(ctx context.Context, serviceName string, conf config.Config) {
	cfg := jaeger.Configuration{
		ServiceName: serviceName,
		Sampler: &jaeger.SamplerConfig{
			Type:  conf.Jaeger.Sampler.Type,
			Param: conf.Jaeger.Sampler.Param,
		},
		Reporter: &jaeger.ReporterConfig{
			LogSpans:           conf.Jaeger.Reporter.LogSpans,
			LocalAgentHostPort: conf.Jaeger.Reporter.LocalAgentHostPort,
			CollectorEndpoint:  conf.Jaeger.Reporter.CollectorEndpoint,
		},
	}
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		zap.L().Fatal("Error initializing Jaeger", zap.Error(err))
	}

	ot.SetGlobalTracer(tracer)
	zap.L().Info("Jaeger has been started")
	<-ctx.Done()

	if err = closer.Close(); err != nil {
		zap.L().Warn("Error shutting down Jaeger", zap.Error(err))
	}
	zap.L().Info("Jaeger has been stopped")
}
