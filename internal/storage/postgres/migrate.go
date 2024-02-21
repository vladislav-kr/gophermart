package postgres

import (
	"context"
	"embed"
	"fmt"
	"log/slog"

	"github.com/pressly/goose/v3"
	"github.com/vladislav-kr/gofermart-bonus/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations
var migrations embed.FS

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetLogger(
		slogGoose(logger.Logger()),
	)
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres migrate set dialect postgres: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("postgres migrate up: %w", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("postgres migrate close db: %w", err)
	}
	return nil
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetLogger(
		slogGoose(logger.Logger()),
	)
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres migrate set dialect postgres: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	if err := goose.DownContext(ctx, db, "migrations"); err != nil {
		return fmt.Errorf("postgres migrate down: %w", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("postgres migrate close db: %w", err)
	}
	return nil
}

var _ goose.Logger = (*slogGooseLogger)(nil)

func slogGoose(l *slog.Logger) goose.Logger {
	return &slogGooseLogger{l: l}
}

type slogGooseLogger struct {
	l *slog.Logger
}

func (l *slogGooseLogger) Fatalf(format string, v ...interface{}) {
	l.l.Error(fmt.Sprintf(format, v...))
}

func (l *slogGooseLogger) Printf(format string, v ...interface{}) {
	l.l.Info(fmt.Sprintf(format, v...))
}
