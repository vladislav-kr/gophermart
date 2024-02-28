package logger

import (
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/go-chi/httplog/v2"
)

type LogLevel string

const (
	LevelLocal LogLevel = "local"
	LevelDev   LogLevel = "dev"
	LevelProd  LogLevel = "prod"
)

var (
	log            *slog.Logger
	httpLog        *httplog.Logger
	httpLogDefault *httplog.Logger
	once           sync.Once
)

type Option func(*loggerOption)

type loggerOption struct {
	level       slog.Level
	writer      io.Writer
	serviceName string
}

func init() {
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	)

	httpLogDefault = httplog.NewLogger("default", httplog.Options{
		JSON:             true,
		LogLevel:         slog.LevelInfo,
		Concise:          true,
		RequestHeaders:   false,
		MessageFieldName: "message",
		Writer:           os.Stdout,
	})

}

func WithServiceName(serviceName string) Option {
	return func(l *loggerOption) {
		l.serviceName = serviceName
	}
}

func WithLevel(lvl LogLevel) Option {
	return func(l *loggerOption) {
		switch lvl {
		case LevelLocal:
			l.level = slog.LevelDebug
		case LevelDev:
			l.level = slog.LevelInfo
		case LevelProd:
			l.level = slog.LevelInfo
		}
	}
}

func WithWriter(w io.Writer) Option {
	return func(l *loggerOption) {
		l.writer = w
	}
}

func apply(opts ...Option) *loggerOption {
	l := loggerOption{
		level:  slog.LevelInfo,
		writer: os.Stdout,
	}

	for _, fn := range opts {
		fn(&l)
	}
	return &l
}

func newLogger(opts ...Option) {
	once.Do(func() {
		l := apply(opts...)

		var handler slog.Handler

		switch l.level {
		case slog.LevelDebug:
			handler = slog.NewTextHandler(l.writer, &slog.HandlerOptions{Level: l.level})
		default:
			handler = slog.NewJSONHandler(l.writer, &slog.HandlerOptions{Level: l.level})
		}

		log = slog.New(handler)

		httpLog = httplog.NewLogger(l.serviceName, httplog.Options{
			JSON:             true,
			LogLevel:         l.level,
			Concise:          true,
			RequestHeaders:   false,
			MessageFieldName: "message",
			Writer:           l.writer,
		})
	})
}

func ConfigureLoggers(opts ...Option) {
	newLogger(opts...)
}

func Logger() *slog.Logger {
	if log == nil {
		return slog.Default()
	}
	return log
}

func HTTPLogger() *httplog.Logger {
	if httpLog == nil {
		return httpLogDefault
	}
	return httpLog
}

func Error(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
