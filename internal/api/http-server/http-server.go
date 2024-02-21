package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/vladislav-kr/gofermart-bonus/internal/logger"
)

type HTTPServer struct {
	log    *slog.Logger
	server *http.Server
}

func NewHTTPServer(
	srv *http.Server,
) *HTTPServer {
	return &HTTPServer{
		log: logger.Logger().With(
			slog.String("addr", srv.Addr),
			slog.String("type", "http"),
		),
		server: srv,
	}
}

func (hs *HTTPServer) Run() error {
	hs.log.Info("running")
	err := hs.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		hs.log.Info("stopped")
		return nil
	}
	return err
}

func (hs *HTTPServer) Stop(ctx context.Context) error {
	hs.log.Info("stopping...")
	return hs.server.Shutdown(ctx)
}
