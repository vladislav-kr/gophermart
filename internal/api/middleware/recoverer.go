package middleware

import (
	"net/http"
	"runtime/debug"

	chimMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/vladislav-kr/gophermart/internal/metrics"
)

func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				metrics.Mertics().PanicInc()

				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}
				logEntry := chimMiddleware.GetLogEntry(r)
				if logEntry != nil {
					logEntry.Panic(rvr, debug.Stack())
				} else {
					chimMiddleware.PrintPrettyStack(rvr)
				}

				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
