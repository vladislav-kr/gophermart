package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vladislav-kr/gophermart/internal/domain/models"
	"github.com/vladislav-kr/gophermart/internal/api/handlers/mocks"
)

func contextWithToken(t *testing.T, userID string) context.Context {
	token, err := jwt.NewBuilder().
		Issuer("gophermart").
		Audience([]string{string(userID)}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(time.Minute*3)).
		Claim("userID", userID).
		Build()

	require.NoError(t, err)

	return context.WithValue(context.Background(), jwtauth.TokenCtxKey, token)
}

func TestHandlers_Login(t *testing.T) {

	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)

	type mockParam struct {
		callMock bool
		cred     models.Credentials
		token    string
		err      error
	}
	type args struct {
		body     string
		handlers *Handlers
		mock     mockParam
	}

	tests := []struct {
		name           string
		args           args
		expectedStatus int
	}{
		{
			name: "пустое тело запроса",
			args: args{
				body:     "",
				handlers: handlers,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "ошибка парсинга json(некорректный json)",
			args: args{
				body:     `{"login": "admin","password": "SuperPassword1234@#!"`,
				handlers: handlers,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "пользователь успешно аутентифицирован",
			args: args{
				body:     `{"login": "admin","password": "SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "admin",
						Password: "SuperPassword1234@#!",
					},
					token: "secret-token",
					err:   nil,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "неверная пара логин/пароль",
			args: args{
				body:     `{"login": "admin","password": "WWW_SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "admin",
						Password: "WWW_SuperPassword1234@#!",
					},
					token: "",
					err:   models.ErrIncorrectCredentials,
				},
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				body:     `{"login": "3admin","password": "3WWW_SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "3admin",
						Password: "3WWW_SuperPassword1234@#!",
					},
					token: "",
					err:   fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(tt.args.body),
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("Login", mock.AnythingOfType("*context.timerCtx"), tt.args.mock.cred).
					Return(tt.args.mock.token, tt.args.mock.err)
			}

			tt.args.handlers.Login(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			if result.StatusCode == http.StatusOK {
				assert.NotEmpty(t, result.Header.Get("Authorization"))
			}
		})
	}
}

func TestHandlers_Register(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)

	type mockParam struct {
		callMock bool
		cred     models.Credentials
		token    string
		err      error
	}
	type args struct {
		body     string
		handlers *Handlers
		mock     mockParam
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
	}{
		{
			name: "пустое тело запроса",
			args: args{
				body:     "",
				handlers: handlers,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "ошибка парсинга json(некорректный json)",
			args: args{
				body:     `{"login": "admin","password": "SuperPassword1234@#!"`,
				handlers: handlers,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "пользователь успешно аутентифицирован",
			args: args{
				body:     `{"login": "admin","password": "SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "admin",
						Password: "SuperPassword1234@#!",
					},
					token: "secret-token",
					err:   nil,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "логин уже занят",
			args: args{
				body:     `{"login": "admin2","password": "SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "admin2",
						Password: "SuperPassword1234@#!",
					},
					token: "",
					err:   models.ErrLoginAlreadyExists,
				},
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				body:     `{"login": "3admin","password": "3WWW_SuperPassword1234@#!"}`,
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					cred: models.Credentials{
						Login:    "3admin",
						Password: "3WWW_SuperPassword1234@#!",
					},
					token: "",
					err:   fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(
				http.MethodPost,
				"/",
				strings.NewReader(tt.args.body),
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("Register", mock.AnythingOfType("*context.timerCtx"), tt.args.mock.cred).
					Return(tt.args.mock.token, tt.args.mock.err)
			}

			tt.args.handlers.Register(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			if result.StatusCode == http.StatusOK {
				assert.NotEmpty(t, result.Header.Get("Authorization"))
			}
		})
	}
}

func TestHandlers_SaveOrder(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)

	userID := models.UserID(uuid.NewString())
	ctx := contextWithToken(t, string(userID))

	type mockParam struct {
		callMock bool
		orderID  models.OrderID
		userID   models.UserID
		err      error
	}
	type args struct {
		ctx      context.Context
		body     string
		handlers *Handlers
		mock     mockParam
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
	}{
		{
			name: "пользователь не аутентифицирован",
			args: args{
				ctx:      context.Background(),
				body:     "2377225624",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "2377225624",
					err:      models.ErrUserIDMandatory,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "неверный формат запроса",
			args: args{
				ctx:      ctx,
				body:     "",
				handlers: handlers,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "новый номер заказа принят в обработку",
			args: args{
				ctx:      ctx,
				body:     "2377225624",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "2377225624",
					userID:   userID,
					err:      nil,
				},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "неверный формат номера заказа",
			args: args{
				ctx:      ctx,
				body:     "237722562499",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "237722562499",
					userID:   userID,
					err:      models.ErrIncorrectOrderNumber,
				},
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "номер заказа уже был загружен этим пользователем",
			args: args{
				ctx:      ctx,
				body:     "order1",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "order1",
					userID:   userID,
					err:      models.ErrAlreadyUploadedUser,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "номер заказа уже был загружен другим пользователем",
			args: args{
				ctx:      ctx,
				body:     "order2",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "order2",
					userID:   userID,
					err:      models.ErrAlreadyUploadedAnotherUser,
				},
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				ctx:      ctx,
				body:     "order3",
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					orderID:  "order3",
					userID:   userID,
					err:      fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(
				tt.args.ctx,
				http.MethodPost,
				"/",
				strings.NewReader(tt.args.body),
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("Order",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.mock.orderID,
					tt.args.mock.userID,
				).
					Return(tt.args.mock.err)
			}

			tt.args.handlers.SaveOrder(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

		})
	}
}

func TestHandlers_ListOrdersByUser(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)
	timeTest, err := time.Parse(time.RFC3339, "2024-02-16T16:16:29.4898414+03:00")
	require.NoError(t, err)

	type mockParam struct {
		callMock bool
		userID   models.UserID
		orders   []models.Order
		err      error
	}
	type args struct {
		ctx      context.Context
		handlers *Handlers
		mock     mockParam
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "пользователь не аутентифицирован",
			args: args{
				ctx:      context.Background(),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					err:      models.ErrUserIDMandatory,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "нет данных для ответа",
			args: args{
				ctx:      contextWithToken(t, "9f059c1c-da6d-4245-9102-d4734a8433db"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "9f059c1c-da6d-4245-9102-d4734a8433db",
					err:      models.ErrNoRecordsFound,
				},
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				ctx:      contextWithToken(t, "5172509d-14b2-4ed0-9dc5-8c8838218426"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "5172509d-14b2-4ed0-9dc5-8c8838218426",
					err:      fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "успешная обработка запроса",
			args: args{
				ctx:      contextWithToken(t, "dd55ca8f-d25f-4242-8d63-06783b69926d"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "dd55ca8f-d25f-4242-8d63-06783b69926d",
					orders: []models.Order{
						{
							OrderID:    "2377225624",
							Status:     "PROCESSED",
							Accrual:    500.5,
							UploadedAt: timeTest,
						},
					},
					err: nil,
				},
			},
			expectedBody:   `[{"number":"2377225624","status":"PROCESSED","uploaded_at":"2024-02-16T16:16:29.4898414+03:00","accrual":500.5}]`,
			expectedStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(
				tt.args.ctx,
				http.MethodGet,
				"/",
				nil,
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("OrdersByUserID",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.mock.userID,
				).
					Return(tt.args.mock.orders, tt.args.mock.err)
			}

			tt.args.handlers.ListOrdersByUser(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			if result.StatusCode == http.StatusOK {
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandlers_BalanceByUser(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)

	type mockParam struct {
		callMock bool
		userID   models.UserID
		balance  *models.Balance
		err      error
	}
	type args struct {
		ctx      context.Context
		handlers *Handlers
		mock     mockParam
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "пользователь не аутентифицирован",
			args: args{
				ctx:      context.Background(),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					err:      models.ErrUserIDMandatory,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				ctx:      contextWithToken(t, "9f059c1c-da6d-4245-9102-d4734a8433db"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "9f059c1c-da6d-4245-9102-d4734a8433db",
					err:      fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "успешная обработка запроса",
			args: args{
				ctx:      contextWithToken(t, "5172509d-14b2-4ed0-9dc5-8c8838218426"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "5172509d-14b2-4ed0-9dc5-8c8838218426",
					balance: &models.Balance{
						Current:   100.43,
						Withdrawn: 394,
					},
					err: nil,
				},
			},
			expectedBody:   `{"current": 100.43,"withdrawn": 394}`,
			expectedStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(
				tt.args.ctx,
				http.MethodGet,
				"/",
				nil,
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("UserBalance",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.mock.userID,
				).
					Return(tt.args.mock.balance, tt.args.mock.err)
			}

			tt.args.handlers.BalanceByUser(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			if result.StatusCode == http.StatusOK {
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandlers_WithdrawBonuses(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)

	type mockParam struct {
		callMock bool
		withdraw models.WithdrawBonuses
		userID   models.UserID
		err      error
	}
	type args struct {
		ctx      context.Context
		body     string
		handlers *Handlers
		mock     mockParam
	}

	tests := []struct {
		name           string
		args           args
		expectedStatus int
	}{
		{
			name: "пользователь не аутентифицирован",
			args: args{
				ctx:      context.Background(),
				handlers: handlers,
				body:     `{"order":"23772256241","sum":300}`,
				mock: mockParam{
					callMock: true,
					withdraw: models.WithdrawBonuses{
						Order: "23772256241",
						Sum:   300,
					},
					err: models.ErrUserIDMandatory,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "неверный формат запроса",
			args: args{
				ctx:      contextWithToken(t, "9f059c1c-da6d-4245-9102-d4734a8433db"),
				handlers: handlers,
				body:     `573885vjv85`,
				mock: mockParam{
					callMock: false,
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "неверный номер заказа",
			args: args{
				ctx:      contextWithToken(t, "9f059c1c-da6d-4245-9102-d4734a8433db"),
				handlers: handlers,
				body:     `{"order":"2377225623","sum":500}`,
				mock: mockParam{
					callMock: true,
					withdraw: models.WithdrawBonuses{
						Order: "2377225623",
						Sum:   500,
					},
					userID: "9f059c1c-da6d-4245-9102-d4734a8433db",
					err:    models.ErrIncorrectOrderNumber,
				},
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				ctx:      contextWithToken(t, "5172509d-14b2-4ed0-9dc5-8c8838218426"),
				handlers: handlers,
				body:     `{"order":"2377225645","sum":700}`,
				mock: mockParam{
					callMock: true,
					withdraw: models.WithdrawBonuses{
						Order: "2377225645",
						Sum:   700,
					},
					userID: "5172509d-14b2-4ed0-9dc5-8c8838218426",
					err:    fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "успешная обработка запроса",
			args: args{
				ctx:      contextWithToken(t, "dd55ca8f-d25f-4242-8d63-06783b69926d"),
				handlers: handlers,
				body:     `{"order":"2377225624","sum":200}`,
				mock: mockParam{
					callMock: true,
					withdraw: models.WithdrawBonuses{
						Order: "2377225624",
						Sum:   200,
					},
					userID: "dd55ca8f-d25f-4242-8d63-06783b69926d",
					err:    nil,
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "на счету недостаточно средств",
			args: args{
				ctx:      contextWithToken(t, "9ac768ed-c871-42e2-9137-20efc6b6b035"),
				handlers: handlers,
				body:     `{"order":"2377225624","sum":300}`,
				mock: mockParam{
					callMock: true,
					withdraw: models.WithdrawBonuses{
						Order: "2377225624",
						Sum:   300,
					},
					userID: "9ac768ed-c871-42e2-9137-20efc6b6b035",
					err:    models.ErrInsufficientFunds,
				},
			},
			expectedStatus: http.StatusPaymentRequired,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(
				tt.args.ctx,
				http.MethodPost,
				"/",
				strings.NewReader(tt.args.body),
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("Withdraw",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.mock.userID,
					tt.args.mock.withdraw,
				).
					Return(tt.args.mock.err)
			}

			tt.args.handlers.WithdrawBonuses(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)
		})
	}
}

func TestHandlers_HistoryWithdrawals(t *testing.T) {
	srv := mocks.NewService(t)
	handlers := NewHandlers(srv, nil)
	timeTest, err := time.Parse(time.RFC3339, "2024-02-16T16:16:29.4898414+03:00")
	require.NoError(t, err)

	type mockParam struct {
		callMock    bool
		userID      models.UserID
		withdrawals []models.WithdrawalsBonuses
		err         error
	}
	type args struct {
		ctx      context.Context
		handlers *Handlers
		mock     mockParam
	}
	tests := []struct {
		name           string
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "пользователь не аутентифицирован",
			args: args{
				ctx:      context.Background(),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					err:      models.ErrUserIDMandatory,
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "внутренняя ошибка сервера",
			args: args{
				ctx:      contextWithToken(t, "9f059c1c-da6d-4245-9102-d4734a8433db"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "9f059c1c-da6d-4245-9102-d4734a8433db",
					err:      fmt.Errorf("failed to connect to the database"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "нет ни одного списания",
			args: args{
				ctx:      contextWithToken(t, "5172509d-14b2-4ed0-9dc5-8c8838218426"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "5172509d-14b2-4ed0-9dc5-8c8838218426",
					err:      models.ErrNoRecordsFound,
				},
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "успешная обработка запроса",
			args: args{
				ctx:      contextWithToken(t, "dd55ca8f-d25f-4242-8d63-06783b69926d"),
				handlers: handlers,
				mock: mockParam{
					callMock: true,
					userID:   "dd55ca8f-d25f-4242-8d63-06783b69926d",
					withdrawals: []models.WithdrawalsBonuses{
						{
							Order:       "2377225624",
							Sum:         300,
							ProcessedAt: timeTest,
						},
					},
					err: nil,
				},
			},
			expectedBody:   `[{"order": "2377225624","sum": 300,"processed_at":"2024-02-16T16:16:29.4898414+03:00"}]`,
			expectedStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(
				tt.args.ctx,
				http.MethodGet,
				"/",
				nil,
			)
			require.NoError(t, err)

			if tt.args.mock.callMock {
				srv.On("WithdrawalsByUserID",
					mock.AnythingOfType("*context.timerCtx"),
					tt.args.mock.userID,
				).
					Return(tt.args.mock.withdrawals, tt.args.mock.err)
			}

			tt.args.handlers.HistoryWithdrawals(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			if result.StatusCode == http.StatusOK {
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tt.expectedBody, string(body))
			}
		})
	}
}

func TestHandlers_Ready(t *testing.T) {
	pinger := mocks.NewPinger(t)
	handlers := NewHandlers(nil, pinger)

	type mockParam struct {
		err error
	}
	type args struct {
		handlers *Handlers
		mock     mockParam
	}

	tests := []struct {
		name           string
		args           args
		expectedStatus int
	}{
		{
			name: "сервис способен обрабатывать запросы",
			args: args{
				handlers: handlers,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "хранилище недоступно",
			args: args{
				handlers: handlers,
				mock: mockParam{
					err: fmt.Errorf("ping db"),
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/", nil)
			require.NoError(t, err)

			pinger.On("Ping", mock.AnythingOfType("*context.timerCtx")).
				Return(tt.args.mock.err).Once()

			tt.args.handlers.Ready(rr, req)

			result := rr.Result()
			defer result.Body.Close()
			assert.Equal(t, tt.expectedStatus, result.StatusCode)
		})
	}

}

func TestHandlers_Live(t *testing.T) {
	handlers := NewHandlers(nil, nil)

	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	handlers.Live(rr, req)

	result := rr.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func Test_userIDFromContext(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name       string
		args       args
		wantUserID string
		wantOk     bool
	}{
		{
			name: "id пользователя найдено",
			args: args{
				ctx: contextWithToken(t, "user1"),
			},
			wantUserID: "user1",
			wantOk:     true,
		},
		{
			name: "id пользователя не найдено",
			args: args{
				ctx: context.Background(),
			},
			wantUserID: "",
			wantOk:     false,
		},
		{
			name: "id пользователя не найдено(userID пустой)",
			args: args{
				ctx: contextWithToken(t, ""),
			},
			wantUserID: "",
			wantOk:     true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			userID, ok := userIDFromContext(tt.args.ctx)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantUserID, userID)
		})
	}
}

func TestHandler_ServeHTTP(t *testing.T) {

	var testError = fmt.Errorf("test error")

	buf := bytes.NewBuffer(make([]byte, 512))

	logger := httplog.NewLogger("httplog", httplog.Options{
		LogLevel:         slog.LevelInfo,
		Concise:          true,
		MessageFieldName: "message",
		TimeFieldFormat:  time.RFC3339,
		Writer:           buf,
	})

	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	httplog.RequestLogger(logger)(
		Handler(
			func(w http.ResponseWriter, r *http.Request) error {
				return testError
			},
		),
	).ServeHTTP(rr, req)

	result := rr.Result()
	defer result.Body.Close()

	logs, err := io.ReadAll(buf)
	require.NoError(t, err)

	assert.True(t, strings.Contains(string(logs), testError.Error()))
}
