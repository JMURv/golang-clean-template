package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/cache/redis"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/hdl/grpc"
	"github.com/JMURv/golang-clean-template/internal/hdl/http"
	"github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	"github.com/JMURv/golang-clean-template/internal/observability/tracing/jaeger"
	"github.com/JMURv/golang-clean-template/internal/repo/db"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"github.com/JMURv/golang-clean-template/internal/smtp"
	"go.uber.org/zap"
)

func mustRegisterLogger(mode string) {
	switch mode {
	case "prod":
		zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	case "dev":
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	}
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			zap.L().Panic("panic occurred", zap.Any("error", err))
		}
	}()

	const path = "config/.env"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf := config.MustLoad(path)
	mustRegisterLogger(conf.Mode)

	go prometheus.New(conf.Server.Port + 5).Start(ctx)
	go jaeger.Start(ctx, conf.ServiceName, conf)

	au := auth.New(conf)
	cache := redis.New(conf)
	repo := db.New(conf)
	svc := ctrl.New(au, repo, cache, s3.New(conf), smtp.New(conf))
	h := http.New(au, svc)
	hg := grpc.New(conf.ServiceName, svc, au)

	go h.Start(conf.Server.Port)
	go hg.Start(conf.Server.GRPCPort)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-c

	zap.L().Info("Shutting down gracefully...")
	if err := h.Close(ctx); err != nil {
		zap.L().Warn("Error closing handler", zap.Error(err))
	}

	if err := hg.Close(); err != nil {
		zap.L().Warn("Error closing grpc handler", zap.Error(err))
	}

	if err := cache.Close(); err != nil {
		zap.L().Warn("Failed to close connection to cache: ", zap.Error(err))
	}

	if err := repo.Close(); err != nil {
		zap.L().Warn("Error closing repository", zap.Error(err))
	}
}
