package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	App struct {
		LogLevel string `env:"APP_LOG_LEVEL" env-default:"prod" env-description:"local, dev, prod"`
	}
	HTTP struct {
		Addr            string        `env:"RUN_ADDRESS" env-description:"адрес и порт запуска сервиса"`
		ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"10s" env-description:"максимальное время ожидания остановки сервера"`
		ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"30s" env-description:"таймаут на чтение"`
		WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"30s" env-description:"таймаут на запись"`
		IdleTimeout     time.Duration `env:"HTTP_IDLE_TIMEOUT" env-default:"90s" env-description:"таймаут простоя подключения"`
	}
	Storage struct {
		Postgres struct {
			URI string `env:"DATABASE_URI" env-description:"адрес подключения к базе данных"`
		}
	}
	Clients struct {
		AccrualSystem struct {
			URI           string        `env:"ACCRUAL_SYSTEM_ADDRESS" env-description:"адрес системы расчёта начислений"`
			RetryCount    int           `env:"ACCRUAL_RETRY_COUNT" env-default:"4" env-description:"кол-во повторов"`
			RetryWaitTime time.Duration `env:"ACCRUAL_RETRY_WAIT_TIME" env-default:"500ms" env-description:"простой между повторами"`
			ReadTimeout   time.Duration `env:"ACCRUAL_READ_TIMEOUT" env-default:"4s" env-description:"таймаут на чтение"`
		}
	}
	Workers struct {
		UpdateOrders struct {
			ReadTimeout  time.Duration `env:"WORKERS_UPDATE_ORDERS_READ_TIMEOUT" env-default:"4s" env-description:"таймаут на чтение"`
			WriteTimeout time.Duration `env:"WORKERS_UPDATE_ORDERS_WRITE_TIMEOUT" env-default:"4s" env-description:"таймаут на запись"`
			ReadLimit    uint32        `env:"WORKERS_UPDATE_ORDERS_READ_LIMIT" env-default:"10" env-description:"лимит чтения заказов для обновления"`
			WorkersLimit uint8         `env:"WORKERS_UPDATE_ORDERS_WORKERS_LIMIT" env-default:"3" env-description:"количество одновременно работающих воркеров"`
		}
	}
}

// Load загружает конфиг
// вернет ошибку, если не существует обязательная env переменная
func Load() (*Config, error) {
	cfg := Config{}
	if err := parseFlags(&cfg); err != nil {
		return nil, err
	}

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func parseFlags(cfg *Config) error {
	flagSet := flag.NewFlagSet("gopher mart", flag.ContinueOnError)

	flagSet.StringVar(
		&cfg.HTTP.Addr,
		"a",
		":3030",
		"адрес и порт запуска сервиса",
	)

	flagSet.StringVar(
		&cfg.Storage.Postgres.URI,
		"d",
		"",
		"адрес подключения к базе данных",
	)

	flagSet.StringVar(
		&cfg.Clients.AccrualSystem.URI,
		"r",
		"",
		"адрес системы расчёта начислений",
	)

	flagSet.Usage = cleanenv.FUsage(flagSet.Output(), cfg, nil, flagSet.Usage)

	return flagSet.Parse(os.Args[1:])
}
