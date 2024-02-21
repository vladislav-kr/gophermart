package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vladislav-kr/gofermart-bonus/internal/clients"
	"github.com/vladislav-kr/gofermart-bonus/internal/domain/models"
	"github.com/vladislav-kr/gofermart-bonus/internal/service/mocks"
	"github.com/vladislav-kr/gofermart-bonus/internal/storage"
)

func testRSAPrivateKey(t *testing.T) *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 2<<10)
	require.NoError(t, err)
	return k
}

func Test_service_Login(t *testing.T) {
	stor := mocks.NewStorage(t)
	gen := mocks.NewPasswordGenerator(t)

	srv := NewService(gen, stor, nil, testRSAPrivateKey(t))

	type mockArgs struct {
		callStorage   bool
		callGenerator bool
		user          *storage.User
		err           error
		errGen        error
	}
	type args struct {
		cred models.Credentials
		mock mockArgs
	}
	tests := []struct {
		name    string
		service *service
		args    args
		want    string
		wantErr error
	}{
		{
			name:    "неправильные учетные данные",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin",
					Password: "1234",
				},
			},
			wantErr: models.ErrIncorrectCredentials,
		},
		{
			name:    "пользователь не существует",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin2",
					Password: "12343434456",
				},
				mock: mockArgs{
					callStorage: true,
					user:        nil,
					err:         storage.ErrNoRecordsFound,
				},
			},
			wantErr: models.ErrIncorrectCredentials,
		},
		{
			name:    "ошибка при обработке запроса",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin3",
					Password: "1239999456",
				},
				mock: mockArgs{
					callStorage: true,
					user:        nil,
					err:         models.ErrInternal,
				},
			},
			want:    "",
			wantErr: models.ErrInternal,
		},
		{
			name:    "пароль не совпадает",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin4",
					Password: "12347777567",
				},
				mock: mockArgs{
					callStorage:   true,
					callGenerator: true,
					user: &storage.User{
						UserID:   "06223dff-1f8f-4430-923f-1072e67e70ce",
						Login:    "mylogin4",
						Password: []byte("12345644447888"),
					},
					err:    nil,
					errGen: models.ErrMismatchedHashAndPassword,
				},
			},
			wantErr: models.ErrIncorrectCredentials,
		},
		{
			name:    "пароль не совпадает(ошибка генератора)",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin5",
					Password: "12345274567",
				},
				mock: mockArgs{
					callStorage:   true,
					callGenerator: true,
					user: &storage.User{
						UserID:   "06223dff-1f8f-4430-923f-1072e67e70ce",
						Login:    "mylogin5",
						Password: []byte("123452367888"),
					},
					err:    nil,
					errGen: fmt.Errorf("compare hash and password"),
				},
			},
			wantErr: models.ErrInternal,
		},
		{
			name:    "успешная аутентификация",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin6",
					Password: "123456666789",
				},
				mock: mockArgs{
					callStorage:   true,
					callGenerator: true,
					user: &storage.User{
						UserID:   "1cf50925-d72d-488b-94e5-426acce77f3c",
						Login:    "mylogin6",
						Password: []byte("123456666789"),
					},
					err: nil,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.callStorage {
				stor.On("User",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.cred.Login,
				).Return(tt.args.mock.user, tt.args.mock.err)
			}
			if tt.args.mock.callGenerator {
				gen.On("CompareHashAndPassword",
					tt.args.mock.user.Password,
					[]byte(tt.args.cred.Password),
				).Return(tt.args.mock.errGen)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			token, err := tt.service.Login(ctx, tt.args.cred)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				assert.Empty(t, token)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, token)

		})
	}
}

func Test_service_Register(t *testing.T) {
	stor := mocks.NewStorage(t)
	gen := mocks.NewPasswordGenerator(t)
	srv := NewService(gen, stor, nil, testRSAPrivateKey(t))

	type mockArgs struct {
		callStorage   bool
		callGenerator bool
		userUUID      string
		passHash      []byte
		err           error
		errGen        error
	}
	type args struct {
		cred models.Credentials
		mock mockArgs
	}
	tests := []struct {
		name    string
		service *service
		args    args
		wantErr error
	}{
		{
			name:    "неправильные учетные данные",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin",
					Password: "12345",
				},
			},
			wantErr: models.ErrIncorrectCredentials,
		},
		{
			name:    "хеш пароля не сгенерировался",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin2",
					Password: "1234566",
				},
				mock: mockArgs{
					callGenerator: true,
					errGen:        fmt.Errorf("generate hash error"),
				},
			},
			wantErr: models.ErrInternal,
		},
		{
			name:    "пользователь уже существует",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin3",
					Password: "123456789",
				},
				mock: mockArgs{
					callGenerator: true,
					passHash:      []byte("123456789"),
					errGen:        nil,
					callStorage:   true,
					err:           storage.ErrUniqueViolation,
				},
			},
			wantErr: models.ErrLoginAlreadyExists,
		},
		{
			name:    "не удалось создать пользователя в бд",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin4",
					Password: "1234567891",
				},
				mock: mockArgs{
					callGenerator: true,
					passHash:      []byte("1234567891"),
					errGen:        nil,
					callStorage:   true,
					err:           fmt.Errorf("db error"),
				},
			},
			wantErr: models.ErrInternal,
		},
		{
			name:    "пользователь успешно зарегистрирован и аутентифицирован",
			service: srv,
			args: args{
				cred: models.Credentials{
					Login:    "mylogin5",
					Password: "12345678912",
				},
				mock: mockArgs{
					callGenerator: true,
					passHash:      []byte("12345678912"),
					callStorage:   true,
					userUUID:      "d183113d-181e-4606-9bc4-09ce19c89f3b",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.callStorage {
				stor.On("CreateUser",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.cred.Login,
					tt.args.mock.passHash,
				).Return(tt.args.mock.userUUID, tt.args.mock.err)
			}
			if tt.args.mock.callGenerator {
				gen.On("GenerateFromPassword",
					[]byte(tt.args.cred.Password),
				).Return(tt.args.mock.passHash, tt.args.mock.errGen)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			token, err := tt.service.Register(ctx, tt.args.cred)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				assert.Empty(t, token)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, token)

		})
	}
}

func Test_service_Order(t *testing.T) {

	stor := mocks.NewStorage(t)
	clnt := mocks.NewAccrual(t)
	srv := NewService(nil, stor, clnt, nil)

	type mockClntOrder struct {
		call         bool
		accrualOrder *clients.OrderAccrual
		err          error
	}

	type mockCreOrd struct {
		call  bool
		order storage.CreateOrder
		err   error
	}

	type mockArg struct {
		// ordByNum  mockOrdByNum
		clntOrder mockClntOrder
		creOrd    mockCreOrd
	}

	type args struct {
		orderID models.OrderID
		userID  models.UserID
		mock    mockArg
	}
	tests := []struct {
		name    string
		service *service
		args    args
		wantErr error
	}{
		{
			name:    "неверный формат номера заказа",
			service: srv,
			args: args{
				orderID: "123456789a3",
				userID:  "fa425e41-5eae-4aa1-b583-8910b48faf7d",
			},
			wantErr: models.ErrIncorrectOrderNumber,
		},
		{
			name:    "неверный id пользователя",
			service: srv,
			args: args{
				orderID: "12345678903",
				userID:  "user_id_123",
			},
			wantErr: models.ErrIncorrectOrderNumber,
		},
		{
			name:    "номер заказа уже был загружен этим пользователем",
			service: srv,
			args: args{
				orderID: "9278923470",
				userID:  "05edaa75-4ef7-4cd7-9a79-587016830a53",
				mock: mockArg{
					clntOrder: mockClntOrder{
						call: true,
						err: fmt.Errorf("client unavailable"),
					},
					creOrd: mockCreOrd{
						call: true,
						order: storage.CreateOrder{
							OrderID: "9278923470",
							Status: "NEW",
						},
						err: storage.ErrAlreadyUploadedUser,
					},
				},
			},
			wantErr: models.ErrAlreadyUploadedUser,
		},
		{
			name:    "номер заказа уже был загружен другим пользователем",
			service: srv,
			args: args{
				orderID: "23772256667",
				userID:  "f6750364-a511-46dc-9bf7-9716f381acfe",
				mock: mockArg{
					clntOrder: mockClntOrder{
						call: true,
						err: fmt.Errorf("client unavailable"),
					},
					creOrd: mockCreOrd{
						call: true,
						order: storage.CreateOrder{
							OrderID: "23772256667",
							Status: "NEW",
						},
						err: storage.ErrAlreadyUploadedAnotherUser,
					},
				},
			},
			wantErr: models.ErrAlreadyUploadedAnotherUser,
		},
		{
			name:    "создание заказа, ошибка бд",
			service: srv,
			args: args{
				orderID: "23772256246",
				userID:  "9c6cdc97-45d8-45fb-aa2d-3e2451cdb343",
				mock: mockArg{
					clntOrder: mockClntOrder{
						call: true,
						accrualOrder: &clients.OrderAccrual{
							Order:   "23772256246",
							Status:  "REGISTERED",
							Accural: 0,
						},
					},
					creOrd: mockCreOrd{
						call: true,
						order: storage.CreateOrder{
							OrderID: "23772256246",
							Status:  "NEW",
							Accrual: 0,
						},
						err: storage.ErrInternal,
					},
				},
			},
			wantErr: models.ErrInternal,
		},
		{
			name:    "новый номер заказа принят в обработку",
			service: srv,
			args: args{
				orderID: "23772256659",
				userID:  "74e8729a-ac5f-4dfe-952f-1ec329877520",
				mock: mockArg{
					clntOrder: mockClntOrder{
						call: true,
						accrualOrder: &clients.OrderAccrual{
							Order:   "23772256659",
							Status:  "REGISTERED",
							Accural: 0,
						},
					},
					creOrd: mockCreOrd{
						call: true,
						order: storage.CreateOrder{
							OrderID: "23772256659",
							Status:  "NEW",
							Accrual: 0,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.args.mock.clntOrder.call {
				clnt.On("Order",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.orderID),
				).Return(
					tt.args.mock.clntOrder.accrualOrder,
					time.Duration(0),
					tt.args.mock.clntOrder.err,
				)
			}
			if tt.args.mock.creOrd.call {
				stor.On("CreateOrder",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.userID),
					tt.args.mock.creOrd.order,
				).Return(
					tt.args.mock.creOrd.err,
				)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			err := tt.service.Order(ctx, tt.args.orderID, tt.args.userID)

			if tt.wantErr != nil {
				assert.ErrorAs(t, tt.wantErr, &err)
				return
			}

			assert.NoError(t, err)

		})
	}
}

func Test_service_OrdersByUserID(t *testing.T) {
	stor := mocks.NewStorage(t)
	srv := NewService(nil, stor, nil, nil)

	type mockArgs struct {
		call   bool
		orders []storage.Order
		err    error
	}
	type args struct {
		userID models.UserID
		mock   mockArgs
	}
	tests := []struct {
		name       string
		service    *service
		args       args
		wantErr    error
		wantOrders []models.Order
	}{
		{
			name:    "некорректный id пользователя",
			service: srv,
			args: args{
				userID: "user_id_1",
			},
			wantErr: models.ErrUserIDMandatory,
		},
		{
			name:    "заказов не найдено",
			service: srv,
			args: args{
				userID: "3a0bdb4b-0dd4-49b6-9ef4-a5f4300f3f3c",
				mock: mockArgs{
					call: true,
					err:  storage.ErrNoRecordsFound,
				},
			},
			wantErr: models.ErrNoRecordsFound,
		},
		{
			name:    "заказы успешно получены",
			service: srv,
			args: args{
				userID: "5cbb01ca-db9a-4ab7-beef-652a7ec89a9d",
				mock: mockArgs{
					call: true,
					orders: []storage.Order{
						{
							OrderID:    "2377225624",
							Status:     "PROCESSED",
							UploadedAt: time.Time{},
							Accrual:    500,
						},
					},
					err: nil,
				},
			},
			wantOrders: []models.Order{
				{
					OrderID:    "2377225624",
					Status:     "PROCESSED",
					UploadedAt: time.Time{},
					Accrual:    500,
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.call {
				stor.On("Orders",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.userID),
				).Return(tt.args.mock.orders, tt.args.mock.err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			orders, err := tt.service.OrdersByUserID(ctx, tt.args.userID)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				assert.Nil(t, orders)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, orders, tt.wantOrders)
		})
	}
}

func Test_service_UserBalance(t *testing.T) {
	stor := mocks.NewStorage(t)
	srv := NewService(nil, stor, nil, nil)
	type mockArgs struct {
		call    bool
		balance *storage.Balance
		err     error
	}
	type args struct {
		userID models.UserID
		mock   mockArgs
	}
	tests := []struct {
		name        string
		service     *service
		args        args
		wantErr     error
		wantBalance *models.Balance
	}{
		{
			name:    "некорректный id пользователя",
			service: srv,
			args: args{
				userID: "user_id_1",
			},
			wantErr: models.ErrUserIDMandatory,
		},
		{
			name:    "данных о балансе не найдено",
			service: srv,
			args: args{
				userID: "3a0bdb4b-0dd4-49b6-9ef4-a5f4300f3f3c",
				mock: mockArgs{
					call: true,
					err:  fmt.Errorf("internal"),
				},
			},
			wantErr: models.ErrInternal,
		},
		{
			name:    "состояние баланса успешно получено",
			service: srv,
			args: args{
				userID: "5cbb01ca-db9a-4ab7-beef-652a7ec89a9d",
				mock: mockArgs{
					call: true,
					balance: &storage.Balance{
						Current:   500,
						Withdrawn: 200,
					},
					err: nil,
				},
			},
			wantBalance: &models.Balance{
				Current:   500,
				Withdrawn: 200,
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.call {
				stor.On("UserBalance",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.userID),
				).Return(tt.args.mock.balance, tt.args.mock.err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			balance, err := tt.service.UserBalance(ctx, tt.args.userID)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				assert.Nil(t, balance)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, balance, tt.wantBalance)
		})
	}
}

func Test_service_WithdrawalsByUserID(t *testing.T) {
	stor := mocks.NewStorage(t)
	srv := NewService(nil, stor, nil, nil)
	type mockArgs struct {
		call        bool
		withdrawals []storage.WithdrawalsBonuses
		err         error
	}
	type args struct {
		userID models.UserID
		mock   mockArgs
	}
	tests := []struct {
		name            string
		service         *service
		args            args
		wantErr         error
		wantWithdrawals []models.WithdrawalsBonuses
	}{
		{
			name:    "некорректный id пользователя",
			service: srv,
			args: args{
				userID: "user_id_1",
			},
			wantErr: models.ErrUserIDMandatory,
		},
		{
			name:    "списания не найдены",
			service: srv,
			args: args{
				userID: "3a0bdb4b-0dd4-49b6-9ef4-a5f4300f3f3c",
				mock: mockArgs{
					call: true,
					err:  storage.ErrNoRecordsFound,
				},
			},
			wantErr: models.ErrNoRecordsFound,
		},
		{
			name:    "списания успешно получены",
			service: srv,
			args: args{
				userID: "c5c38955-edd4-493f-b145-47a66e892580",
				mock: mockArgs{
					call: true,
					withdrawals: []storage.WithdrawalsBonuses{
						{
							Order:       "12345678903",
							Sum:         500,
							ProcessedAt: time.Time{},
						},
					},
				},
			},
			wantWithdrawals: []models.WithdrawalsBonuses{
				{
					Order:       "12345678903",
					Sum:         500,
					ProcessedAt: time.Time{},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.call {
				stor.On("Withdrawals",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.userID),
				).Return(tt.args.mock.withdrawals, tt.args.mock.err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			withdrawals, err := tt.service.WithdrawalsByUserID(ctx, tt.args.userID)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				assert.Nil(t, withdrawals)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, withdrawals, tt.wantWithdrawals)
		})
	}
}

func Test_service_Withdraw(t *testing.T) {
	stor := mocks.NewStorage(t)
	srv := NewService(nil, stor, nil, nil)
	type mockArgs struct {
		call     bool
		withdraw storage.WithdrawBonuses
		err      error
	}
	type args struct {
		userID   models.UserID
		withdraw models.WithdrawBonuses
		mock     mockArgs
	}
	tests := []struct {
		name    string
		service *service
		args    args
		wantErr error
	}{
		{
			name:    "некорректный id пользователя",
			service: srv,
			args: args{
				userID: "user_id_1",
			},
			wantErr: models.ErrUserIDMandatory,
		},
		{
			name:    "некорректный номер заказа",
			service: srv,
			args: args{
				userID: "user_id_1",
				withdraw: models.WithdrawBonuses{
					Order: "123456789",
				},
			},
			wantErr: models.ErrIncorrectOrderNumber,
		},
		{
			name:    "для списания недостаточно средств",
			service: srv,
			args: args{
				userID: "4de614bf-4f57-495f-aa03-71410472e707",
				mock: mockArgs{
					call: true,
					withdraw: storage.WithdrawBonuses{
						Order: "12345678903",
						Sum:   500,
					},
					err: storage.ErrConstraints,
				},
				withdraw: models.WithdrawBonuses{
					Order: "12345678903",
					Sum:   500,
				},
			},
			wantErr: models.ErrInsufficientFunds,
		},
		{
			name:    "бонусы в счет заказа успешно списаны",
			service: srv,
			args: args{
				userID: "0223ea75-5b08-4c03-b130-acfa9ea58ceb",
				mock: mockArgs{
					call: true,
					withdraw: storage.WithdrawBonuses{
						Order: "2377225624",
						Sum:   300,
					},
					err: nil,
				},
				withdraw: models.WithdrawBonuses{
					Order: "2377225624",
					Sum:   300,
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.args.mock.call {
				stor.On("Withdraw",
					mock.AnythingOfType("*context.timerCtx"),
					string(tt.args.userID),
					tt.args.mock.withdraw,
				).Return(tt.args.mock.err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()

			err := tt.service.Withdraw(ctx,
				tt.args.userID,
				tt.args.withdraw)

			if tt.wantErr != nil {
				assert.ErrorAs(t, err, &tt.wantErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}
