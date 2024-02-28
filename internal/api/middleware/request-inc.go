package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/vladislav-kr/gophermart/internal/metrics"
)

func RequestIncMertics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			metrics.Mertics().RequestInc(ww.Status())
			metrics.Mertics().RequestStatusInc(r.Method, r.RequestURI, ww.Status())
		}()
		next.ServeHTTP(ww, r)
	})
}
