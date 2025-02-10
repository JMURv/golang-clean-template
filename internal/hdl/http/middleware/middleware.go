package middleware

import (
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	"github.com/JMURv/golang-clean-template/internal/hdl/http/utils"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"net/http"
)

func ApplyMiddleware(h http.HandlerFunc, middleware ...func(http.Handler) http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var handler http.Handler = h
		for _, m := range middleware {
			handler = m(handler)
		}
		handler.ServeHTTP(w, r)
	}
}

func RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					zap.L().Error("panic", zap.Any("err", err))
					utils.ErrResponse(w, http.StatusInternalServerError, ctrl.ErrInternalError)
				}
			}()
			next.ServeHTTP(w, r)
		},
	)
}

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			span := opentracing.GlobalTracer().StartSpan(
				fmt.Sprintf("%s %s", r.Method, r.URL),
			)
			defer span.Finish()

			zap.L().Info("Request", zap.String("method", r.Method), zap.String("uri", r.RequestURI))
			next.ServeHTTP(w, r)
		},
	)
}
