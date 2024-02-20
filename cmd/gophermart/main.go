package main

import (
	"context"
	"log/slog"

	"github.com/vladislav-kr/gofermart-bonus/internal/app"
	"github.com/vladislav-kr/gofermart-bonus/internal/config"
	"github.com/vladislav-kr/gofermart-bonus/internal/logger"
)

func main() {

	cfg := config.MustLoad()

	logger.ConfigureLoggers(
		logger.WithLevel(logger.LogLevel(cfg.App.LogLevel)),
		logger.WithServiceName("gofermart"),
	)

	log := logger.Logger().
		With("app", "gofermart").
		With("component", "main")

	log.Info("starting...")
	log.Debug("debug mode", slog.Any("config", cfg))

	// Основной контекст приложения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.NewApp(
		app.Option{
			HTTP: app.HTTP{
				Host:            cfg.HTTP.Addr,
				ReadTimeout:     cfg.HTTP.ReadTimeout,
				WriteTimeout:    cfg.HTTP.WriteTimeout,
				IdleTimeout:     cfg.HTTP.IdleTimeout,
				ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
			},
			Clients: app.Clients{
				Accrual: app.AccrualSystem{
					URI:           cfg.Clients.AccrualSystem.URI,
					RetryCount:    cfg.Clients.AccrualSystem.RetryCount,
					RetryWaitTime: cfg.Clients.AccrualSystem.RetryWaitTime,
					ReadTimeout:   cfg.Clients.AccrualSystem.ReadTimeout,
				},
			},
			Storages: app.Storages{
				Postgres: app.PostgresStorage{
					URI: cfg.Storage.Postgres.URI,
				},
			},
			Workers: app.Workers{
				UpdateOrders: app.WorkerUpdateOrdes{
					ReadTimeout:  cfg.Workers.UpdateOrders.ReadTimeout,
					WriteTimeout: cfg.Workers.UpdateOrders.WriteTimeout,
					ReadLimit:    cfg.Workers.UpdateOrders.ReadLimit,
					WorkersLimit: cfg.Workers.UpdateOrders.WorkersLimit,
				},
			},
		},
	).Run(ctx); err != nil {
		log.Error(err.Error())
	}

}
