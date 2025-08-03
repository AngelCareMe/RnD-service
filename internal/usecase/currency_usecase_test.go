package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"RnD-service/internal/entity"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCurrencyService struct {
	mock.Mock
}

func (m *mockCurrencyService) StoreRatesFromCbr(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockCurrencyService) GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error) {
	args := m.Called(ctx, charCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Currency), args.Error(1)
}

func (m *mockCurrencyService) GetRateByCharCodeAndDate(ctx context.Context, charCode string, date time.Time) (*entity.Currency, error) {
	args := m.Called(ctx, charCode, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Currency), args.Error(1)
}

func setupTestUsecase() (*CurrencyUsecase, *mockCurrencyService, *logrus.Logger, *test.Hook) {
	mockService := new(mockCurrencyService)
	logger, hook := test.NewNullLogger()
	usecase := NewCurrencyUsecase(mockService, logger)
	return usecase, mockService, logger, hook
}

func TestFetchAndStoreRatesFromCBR(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	mockService.On("StoreRatesFromCbr", ctx).Return(nil)

	err := usecase.FetchAndStoreRatesFromCBR(ctx)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestFetchAndStoreRatesFromCBR_Error(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	expectedErr := errors.New("service error")
	mockService.On("StoreRatesFromCbr", ctx).Return(expectedErr)

	err := usecase.FetchAndStoreRatesFromCBR(ctx)
	assert.Equal(t, expectedErr, err)

	mockService.AssertExpectations(t)
}

func TestGetRateByCharCode_InvalidCode(t *testing.T) {
	ctx := context.Background()
	usecase, _, _, _ := setupTestUsecase()

	charCode := "us"
	amount := 1.0

	_, err := usecase.GetRateByCharCode(ctx, charCode, amount)
	assert.ErrorContains(t, err, "invalid char code format")
}

func TestGetRateByCharCode_ServiceError(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	charCode := "USD"
	amount := 1.0

	expectedErr := errors.New("service error")
	mockService.On("GetRateByCharCode", ctx, charCode).Return((*entity.Currency)(nil), expectedErr)

	_, err := usecase.GetRateByCharCode(ctx, charCode, amount)
	assert.Equal(t, expectedErr, err)

	mockService.AssertExpectations(t)
}

func TestGetRateByCharCode_Success(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	charCode := "usd"
	amount := 2.0
	currency := &entity.Currency{
		CharCode: "USD",
		Nominal:  1,
		Value:    90.5,
	}

	mockService.On("GetRateByCharCode", ctx, "USD").Return(currency, nil)

	result, err := usecase.GetRateByCharCode(ctx, charCode, amount)
	assert.NoError(t, err)
	assert.Equal(t, "USD", result.CharCode)
	assert.Equal(t, 181.0, result.ValueRUB)

	mockService.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_InvalidCode(t *testing.T) {
	ctx := context.Background()
	usecase, _, _, _ := setupTestUsecase()

	charCode := "us"
	date := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := 1.0

	_, err := usecase.GetHistoricalRateByCharCode(ctx, charCode, date, amount)
	assert.ErrorContains(t, err, "invalid char code format")
}

func TestGetHistoricalRateByCharCode_FutureDate(t *testing.T) {
	ctx := context.Background()
	usecase, _, _, _ := setupTestUsecase()

	charCode := "USD"
	date := time.Now().Add(24 * time.Hour)
	amount := 1.0

	_, err := usecase.GetHistoricalRateByCharCode(ctx, charCode, date, amount)
	assert.ErrorContains(t, err, "cannot fetch rates for future dates")
}

func TestGetHistoricalRateByCharCode_ZeroDate(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	charCode := "USD"
	var date time.Time
	amount := 1.0
	today := time.Now().Truncate(24 * time.Hour)
	currency := &entity.Currency{
		CharCode: "USD",
		Nominal:  1,
		Value:    90.5,
		Date:     time.Now(),
	}

	mockService.On("GetRateByCharCodeAndDate", ctx, "USD", today).Return(currency, nil)

	result, err := usecase.GetHistoricalRateByCharCode(ctx, charCode, date, amount)
	assert.NoError(t, err)
	assert.Equal(t, "USD", result.CharCode)
	assert.Equal(t, 90.5, result.ValueRUB)

	mockService.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_ServiceError(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	charCode := "USD"
	date := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := 1.0

	expectedErr := errors.New("service error")
	mockService.On("GetRateByCharCodeAndDate", ctx, "USD", date).Return((*entity.Currency)(nil), expectedErr)

	_, err := usecase.GetHistoricalRateByCharCode(ctx, charCode, date, amount)
	assert.Equal(t, expectedErr, err)

	mockService.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_Success(t *testing.T) {
	ctx := context.Background()
	usecase, mockService, _, _ := setupTestUsecase()

	charCode := "USD"
	date := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := 2.0
	currency := &entity.Currency{
		CharCode: "USD",
		Nominal:  1,
		Value:    90.5,
		Date:     date,
	}

	mockService.On("GetRateByCharCodeAndDate", ctx, "USD", date).Return(currency, nil)

	result, err := usecase.GetHistoricalRateByCharCode(ctx, charCode, date, amount)
	assert.NoError(t, err)
	assert.Equal(t, "USD", result.CharCode)
	assert.Equal(t, 181.0, result.ValueRUB)

	mockService.AssertExpectations(t)
}
