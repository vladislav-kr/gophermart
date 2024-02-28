package postgres

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/vladislav-kr/gophermart/internal/storage"
)

type testConfig struct {
	Host           string
	Port           uint16
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
	Username       string
	Password       string
	DBName         string
}

func (c testConfig) connectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.DBName)
}

type testStorager interface {
	storage.Storage
	clean(ctx context.Context) error
}

type PostgresTestSuite struct {
	suite.Suite
	testStorager
	tc  *tcpostgres.PostgresContainer
	cfg *testConfig
}

func (ts *PostgresTestSuite) SetupSuite() {
	cfg := &testConfig{
		ConnectTimeout: 5 * time.Second,
		QueryTimeout:   5 * time.Second,
		Username:       "postgres",
		Password:       "password",
		DBName:         "postgres",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pgc, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("docker.io/postgres:latest"),
		tcpostgres.WithDatabase(cfg.DBName),
		tcpostgres.WithUsername(cfg.Username),
		tcpostgres.WithPassword(cfg.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	// if err != nil {
	// 	ts.T().Skipf("skip docker in docker")
	// }
	require.NoError(ts.T(), err)

	cfg.Host, err = pgc.Host(ctx)
	require.NoError(ts.T(), err)

	port, err := pgc.MappedPort(ctx, "5432")
	require.NoError(ts.T(), err)

	cfg.Port = uint16(port.Int())

	ts.tc = pgc
	ts.cfg = cfg

	db, err := New(ctx, Config{
		URI: cfg.connectionString(),
	})
	require.NoError(ts.T(), err)

	ts.testStorager = db

	ts.T().Logf("stared postgres at %s:%d", cfg.Host, cfg.Port)

}

func (ts *PostgresTestSuite) TearDownSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(ts.T(), ts.clean(ctx))
	require.NoError(ts.T(), ts.Close())
	require.NoError(ts.T(), ts.tc.Terminate(ctx))
}

func TestPostgres(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

func (s *dbStorage) clean(ctx context.Context) error {
	newCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	return migrateDown(newCtx, s.pool)
}

func (ts *PostgresTestSuite) SetupTest() {}

func (ts *PostgresTestSuite) TearDownTest() {}

func (ts *PostgresTestSuite) TestPing() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ts.NoError(ts.Ping(ctx))
}

// создание и получение пользователя
func (ts *PostgresTestSuite) TestUser() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	login1 := "userlogin1"
	pass1 := []byte("secret1")
	//создание нового пользователя
	userID, err := ts.CreateUser(context.Background(), login1, pass1)
	ts.NoError(err)
	user, err := ts.User(ctx, login1)
	ts.NoError(err)
	ts.Equal(user.UserID, userID)
	ts.Equal(user.Password, pass1)

	//проверка уникальности пользователей
	_, err = ts.CreateUser(ctx, login1, pass1)
	ts.ErrorIs(err, storage.ErrUniqueViolation)

	//пользователь не существует
	_, err = ts.User(ctx, "no-name")
	ts.ErrorIs(err, storage.ErrNoRecordsFound)
}

// создание заказа и проверка баланса
func (ts *PostgresTestSuite) TestCreatingOrderCheckBalance() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	userID, err := ts.CreateUser(ctx, "user-create-order-1", []byte("secret"))
	ts.NoError(err)

	ord := storage.CreateOrder{
		OrderID: "create-order-1",
		Status:  "PROCESSED",
		Accrual: 50.55,
	}

	err = ts.CreateOrder(ctx, userID, ord)
	ts.NoError(err)

	err = ts.CreateOrder(ctx, userID, ord)
	ts.ErrorIs(err, storage.ErrAlreadyUploadedUser)

	userID2, err := ts.CreateUser(ctx, "user-create-order-2", []byte("secret"))
	ts.NoError(err)

	err = ts.CreateOrder(ctx, userID2, ord)
	ts.ErrorIs(err, storage.ErrAlreadyUploadedAnotherUser)

	balance, err := ts.UserBalance(ctx, userID)
	ts.NoError(err)
	ts.Equal(ord.Accrual, balance.Current)
}

// списание со счета и проверка истории
func (ts *PostgresTestSuite) TestWithdraw() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	userID, err := ts.CreateUser(ctx, "user-withdraw", []byte("secret"))
	ts.NoError(err)

	ord := storage.CreateOrder{
		OrderID: "withdraw-order-1",
		Status:  "PROCESSED",
		Accrual: 500.5,
	}

	err = ts.CreateOrder(ctx, userID, ord)
	ts.NoError(err)

	withdraw := storage.WithdrawBonuses{
		Order: "withdraw-order-2",
		Sum:   500,
	}

	err = ts.Withdraw(ctx, userID, withdraw)
	ts.NoError(err)

	withdrawals, err := ts.Withdrawals(ctx, userID)
	ts.NoError(err)
	ts.Require().Equal(len(withdrawals), 1)
	ts.Equal(withdraw.Order, withdrawals[0].Order)
	ts.Equal(withdraw.Sum, withdrawals[0].Sum)

	withdraw = storage.WithdrawBonuses{
		Order: "withdraw-order-3",
		Sum:   500,
	}
	// недостаточно средств
	err = ts.Withdraw(ctx, userID, withdraw)
	ts.ErrorIs(err, storage.ErrConstraints)
}

// получение заказов пользователя
func (ts *PostgresTestSuite) TestReadOrders() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	userID, err := ts.CreateUser(ctx, "user-read-orders", []byte("secret"))
	ts.Require().NoError(err)

	err = ts.CreateOrder(ctx, userID, storage.CreateOrder{
		OrderID: "read-orders-1",
		Status:  "PROCESSED",
		Accrual: 100,
	})
	ts.Require().NoError(err)
	err = ts.CreateOrder(ctx, userID, storage.CreateOrder{
		OrderID: "read-orders-2",
		Status:  "PROCESSED",
		Accrual: 200,
	})
	ts.Require().NoError(err)

	orders, err := ts.Orders(ctx, userID)
	ts.Require().NoError(err)
	ts.Equal(len(orders), 2)

}

// массовое получение заказов и их обновление
func (ts *PostgresTestSuite) TestReadOrdersAndBatchUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	userID, err := ts.CreateUser(ctx, "user-batch-update", []byte("secret"))
	ts.NoError(err)

	createOrders := []storage.CreateOrder{
		{OrderID: "readupdate1", Status: "NEW"},
		{OrderID: "readupdate2", Status: "PROCESSING"},
		{OrderID: "readupdate3", Status: "PROCESSING"},
	}

	//заказы для теста
	for _, ord := range createOrders {
		err = ts.CreateOrder(ctx, userID, ord)
		ts.Require().NoError(err)
	}

	//получить заказы для обновления
	orders, err := ts.OrdersForUpdate(ctx, 0)
	ts.Require().NoError(err)

	updateOrder := make([]storage.UpdateOrder, 0, len(orders))

	for _, ord := range orders {
		updateOrder = append(updateOrder, storage.UpdateOrder{
			UserID:  userID,
			OrderID: ord.OrderID,
			Status:  "PROCESSED",
			Accrual: 100,
		})
	}

	//сохранит заказы с новыми данными
	err = ts.BatchUpdateOrder(ctx, updateOrder)
	ts.Require().NoError(err)

	//все заказы из бд
	updatedOrders, err := ts.Orders(ctx, userID)
	ts.Require().NoError(err)

	slices.SortFunc[[]storage.UpdateOrder, storage.UpdateOrder](updateOrder,
		func(a, b storage.UpdateOrder) int {
			if a.OrderID < b.OrderID {
				return -1
			}
			if a.OrderID == b.OrderID {
				return 0
			}
			return 1
		})

	slices.SortFunc[[]storage.Order, storage.Order](updatedOrders,
		func(a, b storage.Order) int {
			if a.OrderID < b.OrderID {
				return -1
			}
			if a.OrderID == b.OrderID {
				return 0
			}
			return 1
		})

	equal := slices.EqualFunc[[]storage.UpdateOrder, []storage.Order](updateOrder, updatedOrders,
		func(uo storage.UpdateOrder, o storage.Order) bool {
			return uo.OrderID == o.OrderID &&
				uo.Status == o.Status &&
				uo.Accrual == uo.Accrual
		})

	ts.True(equal, "обработанные заказы для сохранение, не равны фактически сохраненным заказам")

}
