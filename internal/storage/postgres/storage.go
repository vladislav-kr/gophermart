package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladislav-kr/gofermart-bonus/internal/logger"
	"github.com/vladislav-kr/gofermart-bonus/internal/storage"
)

var _ storage.Storage = (*dbStorage)(nil)

type Config struct {
	URI string
}

type dbStorage struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func (s *dbStorage) Close() error {
	s.pool.Close()
	return nil
}

func (s *dbStorage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func New(ctx context.Context, cfg Config) (*dbStorage, error) {

	pool, err := pgxpool.New(ctx, cfg.URI)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if err := migrate(ctx, pool); err != nil {
		return nil, err
	}

	return &dbStorage{
		pool: pool,
		log: logger.Logger().With(
			slog.String("component", "storage"),
		),
	}, nil
}

func (s *dbStorage) User(ctx context.Context, login string) (*storage.User, error) {

	user := storage.User{}

	query := `
		SELECT
			user_id,
			login,
			pass_hash
		FROM
			users
		WHERE
			login = @login
		LIMIT 1`

	args := pgx.NamedArgs{"login": login}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query uesrs by login %s: %w", login, err)
	}

	if user, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[storage.User]); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, storage.ErrNoRecordsFound
		default:
			return nil, fmt.Errorf("collect one row: %w", err)
		}
	}

	return &user, nil
}

func (s *dbStorage) CreateUser(ctx context.Context,
	login string,
	passwordHash []byte,
) (string, error) {

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.log.Error(
				"rolback create user",
				logger.Error(err),
			)
		}
	}()

	userUUID := uuid.NewString()

	queryUser := `
		INSERT INTO
			users (user_id, login, pass_hash)
		VALUES
			(@userID, @login, @passwordHash)`

	argsUser := pgx.NamedArgs{
		"userID":       userUUID,
		"login":        login,
		"passwordHash": passwordHash,
	}

	if _, err := tx.Exec(ctx, queryUser, argsUser); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgerrcode.UniqueViolation {
			return "", fmt.Errorf("login %s: %w", login, storage.ErrUniqueViolation)
		}
	}

	queryUserBalance := `
		INSERT INTO
			user_balance (user_id)
		VALUES
			(@userID)`

	argsUserBalance := pgx.NamedArgs{
		"userID": userUUID,
	}

	if _, err := tx.Exec(ctx, queryUserBalance, argsUserBalance); err != nil {
		return "", fmt.Errorf("user_balance insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit work create user: %w", err)
	}

	return userUUID, nil
}

func (s *dbStorage) OrderByNumber(
	ctx context.Context,
	orderID string,
) (*storage.Order, error) {

	query := `
		SELECT
			order_id,
			user_id,
			status,
			uploaded_at,
			changed_at,
			accrual
		FROM
			orders
		WHERE
			order_id = @orderID
		LIMIT 1`

	args := pgx.NamedArgs{"orderID": orderID}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query orders by orderID %s %v: %w",
			orderID,
			err,
			storage.ErrInternal,
		)
	}

	order, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[storage.Order])
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, storage.ErrNoRecordsFound
		default:
			return nil, fmt.Errorf("collect one row %v: %w", err, storage.ErrInternal)
		}
	}

	return &order, nil

}

func (s *dbStorage) CreateOrder(ctx context.Context, userID string, order storage.CreateOrder) error {

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.log.Error("transaction create order rollback", logger.Error(err))
		}
	}()

	query := `
		INSERT INTO
			orders (order_id, user_id, status, accrual)
		VALUES
			(@orderID, @userID, @status, @accrual)`

	args := pgx.NamedArgs{
		"orderID": order.OrderID,
		"userID":  userID,
		"status":  order.Status,
		"accrual": order.Accrual,
	}

	if _, err := tx.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("orders insert, %v: %w", err, storage.ErrConstraints)
	}

	if order.Accrual > 0 {
		queryBalance := `
			UPDATE user_balance balance
			SET
				(current) = (
					SELECT
						current + @accrual
					FROM
						user_balance current_balance
					WHERE
						current_balance.user_id = balance.user_id
				)
			WHERE
				balance.user_id = @userID`

		argsBalance := pgx.NamedArgs{
			"userID":  userID,
			"accrual": order.Accrual,
		}

		if _, err := tx.Exec(ctx, queryBalance, argsBalance); err != nil {
			return fmt.Errorf("user_balance update, %v: %w", err, storage.ErrConstraints)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaction create order commit: %w", err)
	}

	return nil
}

func (s *dbStorage) BatchUpdateOrder(ctx context.Context, orders []storage.UpdateOrder) error {
	query := `
		UPDATE orders
		SET
			status = @status,
			accrual = @accrual,
			changed_at = CURRENT_TIMESTAMP
		WHERE
			order_id = @orderID;`

	batch := &pgx.Batch{}

	for _, order := range orders {
		batch.Queue(query, pgx.NamedArgs{
			"orderID": order.OrderID,
			"status":  order.Status,
			"accrual": order.Accrual,
		})
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	errs := make([]error, 0)
	for i := 0; i < len(orders); i++ {
		if _, err := results.Exec(); err != nil {
			errs = append(errs, fmt.Errorf("update order: %w", err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("batch results close: %w", err)
	}

	batchBalance := &pgx.Batch{}

	queryBalance := `
		UPDATE user_balance balance
		SET
			(current) = (
				SELECT
					current + @accrual
				FROM
					user_balance current_balance
				WHERE
					current_balance.user_id = balance.user_id
			)
		WHERE
			balance.user_id = @userID`

	for _, order := range orders {
		batchBalance.Queue(queryBalance, pgx.NamedArgs{
			"userID":  order.UserID,
			"accrual": order.Accrual,
		})
	}

	resultsBalance := s.pool.SendBatch(ctx, batchBalance)
	defer resultsBalance.Close()

	errsBalance := make([]error, 0)
	for i := 0; i < len(orders); i++ {
		if _, err := resultsBalance.Exec(); err != nil {
			errsBalance = append(errsBalance, fmt.Errorf("update user balance: %w", err))
		}
	}
	if len(errsBalance) > 0 {
		return errors.Join(errsBalance...)
	}

	if err := resultsBalance.Close(); err != nil {
		return fmt.Errorf("batch user balance results close: %w", err)
	}

	return nil
}

func (s *dbStorage) OrdersForUpdate(
	ctx context.Context,
	limit uint32,
) ([]storage.UpdateOrderID, error) {

	if limit == 0 {
		limit = 100
	}

	query := `
		SELECT
			user_id,
			order_id
		FROM
			orders
		WHERE
			status IN ('PROCESSING', 'NEW')
		ORDER BY
			uploaded_at
		LIMIT
			@limit`

	args := pgx.NamedArgs{"limit": limit}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query orders for update: %w", err)
	}

	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[storage.UpdateOrderID])
	if err != nil {
		return nil, fmt.Errorf("collect rows orders: %w", err)
	}

	if len(orders) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return orders, nil
}

func (s *dbStorage) Orders(ctx context.Context, userID string) ([]storage.Order, error) {
	query := `
		SELECT
			order_id,
			user_id,
			status,
			uploaded_at,
			changed_at,
			accrual
		FROM
			orders
		WHERE
			user_id = @userID
		ORDER BY
			uploaded_at`

	args := pgx.NamedArgs{"userID": userID}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query orders by userID %s: %w", userID, err)
	}

	orders, err := pgx.CollectRows(rows, pgx.RowToStructByName[storage.Order])
	if err != nil {
		return nil, fmt.Errorf("collect rows orders: %w", err)
	}

	if len(orders) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return orders, nil
}

func (s *dbStorage) UserBalance(
	ctx context.Context,
	userID string,
) (*storage.Balance, error) {
	query := `
		SELECT
			current,
			withdrawn
		FROM
			user_balance
		WHERE
			user_id = @userID`

	args := pgx.NamedArgs{"userID": userID}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query user_balance by userID %s: %w", userID, err)
	}

	balance, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[storage.Balance])
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, storage.ErrNoRecordsFound
		default:
			return nil, fmt.Errorf("collect rows user_balance: %w", err)
		}
	}

	return &balance, nil
}

func (s *dbStorage) Withdrawals(ctx context.Context, userID string) ([]storage.WithdrawalsBonuses, error) {
	query := `
		SELECT
			order_id,
			sum,
			processed_at
		FROM
			withdrawals
		WHERE
			user_id = @userID`

	args := pgx.NamedArgs{"userID": userID}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("query withdrawals by userID %s: %w", userID, err)
	}

	withdrawals, err := pgx.CollectRows(rows, pgx.RowToStructByName[storage.WithdrawalsBonuses])
	if err != nil {
		return nil, fmt.Errorf("collect rows withdrawals: %w", err)
	}

	if len(withdrawals) == 0 {
		return nil, storage.ErrNoRecordsFound
	}

	return withdrawals, nil
}

func (s *dbStorage) Withdraw(
	ctx context.Context,
	userID string,
	withdraw storage.WithdrawBonuses,
) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.log.Error("transaction withdraw rollback", logger.Error(err))
		}
	}()

	queryBalance := `
		UPDATE user_balance balance
		SET
			(current, withdrawn) = (
				SELECT
					current - @current,
					withdrawn + @withdrawn
				FROM
					user_balance current_balance
				WHERE
					current_balance.user_id = balance.user_id
			)
		WHERE
			balance.user_id = @userID`

	argsBalance := pgx.NamedArgs{
		"userID":    userID,
		"current":   withdraw.Sum,
		"withdrawn": withdraw.Sum,
	}

	if _, err := tx.Exec(ctx, queryBalance, argsBalance); err != nil {
		return fmt.Errorf("user_balance update, %v: %w", err, storage.ErrConstraints)
	}

	queryWithdraw := `
		INSERT INTO
			withdrawals (user_id, order_id, sum)
		VALUES
			(@userID, @orderID, @sum)`

	argsWithdraw := pgx.NamedArgs{
		"userID":  userID,
		"orderID": withdraw.Order,
		"sum":     withdraw.Sum,
	}

	if _, err := tx.Exec(ctx, queryWithdraw, argsWithdraw); err != nil {
		return fmt.Errorf("withdrawals insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaction withdraw commit: %w", err)
	}

	return nil
}
