package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	metrics "github.com/JMURv/golang-clean-template/internal/observability/metrics/prometheus"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

func Auth(au auth.Core) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				access, err := r.Cookie(config.AccessCookieName)
				if err != nil {
					if errors.Is(err, http.ErrNoCookie) {
						utils.ErrResponse(w, http.StatusUnauthorized, err)
						return
					} else {
						zap.L().Error("failed to get access cookie", zap.Error(err))
						utils.ErrResponse(w, http.StatusInternalServerError, err)
						return
					}
				}

				claims, err := au.ParseClaims(r.Context(), access.Value)
				if err != nil {
					utils.ErrResponse(w, http.StatusForbidden, err)
					return
				}

				ctx := context.WithValue(r.Context(), "uid", claims.UID)
				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{w, http.StatusOK}
}

func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func Prometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			s := time.Now()
			op := fmt.Sprintf("%s %s", r.Method, r.RequestURI)

			lrw := NewLoggingResponseWriter(w)
			next.ServeHTTP(lrw, r)
			metrics.ObserveRequest(time.Since(s), lrw.statusCode, op)
		},
	)
}

func Logger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()
				lrw := NewLoggingResponseWriter(w)
				logger.Debug(
					"-->",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote", r.RemoteAddr),
				)

				next.ServeHTTP(lrw, r)

				logger.Info(
					"<--",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", lrw.statusCode),
					zap.Duration("duration", time.Since(start)),
					zap.String("remote", r.RemoteAddr),
				)
			},
		)
	}
}

func OT(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			span, ctx := opentracing.StartSpanFromContext(r.Context(), fmt.Sprintf("%s %s", r.Method, r.RequestURI))
			defer span.Finish()

			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
