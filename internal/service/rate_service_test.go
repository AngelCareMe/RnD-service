package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"RnD-service/internal/adapter/cbr"
	"RnD-service/internal/adapter/postgres"
	"RnD-service/internal/entity"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCbrClient struct {
	mock.Mock
}

func (m *mockCbrClient) FetchRates(ctx context.Context, date string) (*cbr.ValCurs, error) {
	args := m.Called(ctx, date)
	return args.Get(0).(*cbr.ValCurs), args.Error(1)
}

type mockPostgresRepo struct {
	mock.Mock
}

func (m *mockPostgresRepo) StoreRates(ctx context.Context, rates []entity.Currency) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

func (m *mockPostgresRepo) GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error) {
	args := m.Called(ctx, charCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Currency), args.Error(1)
}

func (m *mockPostgresRepo) StoreHistoricalRates(ctx context.Context, date time.Time, rates []entity.Currency) error {
	args := m.Called(ctx, date, rates)
	return args.Error(0)
}

func (m *mockPostgresRepo) GetRateByCharCodeAndDate(ctx context.Context, charCode, date string) (*entity.Currency, error) {
	args := m.Called(ctx, charCode, date)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Currency), args.Error(1)
}

func setupTestService() (*RateService, *mockCbrClient, *mockPostgresRepo, *logrus.Logger, *test.Hook) {
	mockCbr := new(mockCbrClient)
	mockRepo := new(mockPostgresRepo)
	logger, hook := test.NewNullLogger()
	service := NewRateService(mockCbr, mockRepo, logger)
	return service, mockCbr, mockRepo, logger, hook
}

func TestStoreRatesFromCbr(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, mockRepo, _, _ := setupTestService()

	dateStr := time.Now().Format("02/01/2006")
	sampleResp := &cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "USD", Name: "US Dollar", Nominal: 1, Value: "90.5", NumCode: "840"},
			{CharCode: "EUR", Name: "Euro", Nominal: 1, Value: "100.2", NumCode: "978"},
		},
		Date: time.Now().Format("02.01.2006"),
	}

	mockCbr.On("FetchRates", ctx, dateStr).Return(sampleResp, nil)

	rates, err := convertCBRResponse(*sampleResp)
	require.NoError(t, err)

	mockRepo.On("StoreRates", ctx, mock.MatchedBy(func(r []entity.Currency) bool {
		return assert.ElementsMatch(t, rates, r)
	})).Return(nil)

	err = service.StoreRatesFromCbr(ctx)
	assert.NoError(t, err)

	mockCbr.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestStoreRatesFromCbr_FetchError(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, _, _, _ := setupTestService()

	dateStr := time.Now().Format("02/01/2006")
	expectedErr := errors.New("fetch error")
	mockCbr.On("FetchRates", ctx, dateStr).Return((*cbr.ValCurs)(nil), expectedErr)

	err := service.StoreRatesFromCbr(ctx)
	assert.ErrorContains(t, err, expectedErr.Error())

	mockCbr.AssertExpectations(t)
}

func TestStoreRatesFromCbr_StoreError(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, mockRepo, _, _ := setupTestService()

	dateStr := time.Now().Format("02/01/2006")
	sampleResp := &cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "USD", Name: "US Dollar", Nominal: 1, Value: "90.5", NumCode: "840"},
		},
		Date: time.Now().Format("02.01.2006"),
	}

	mockCbr.On("FetchRates", ctx, dateStr).Return(sampleResp, nil)

	rates, err := convertCBRResponse(*sampleResp)
	require.NoError(t, err)

	expectedErr := errors.New("store error")
	mockRepo.On("StoreRates", ctx, mock.MatchedBy(func(r []entity.Currency) bool {
		return assert.ElementsMatch(t, rates, r)
	})).Return(expectedErr)

	err = service.StoreRatesFromCbr(ctx)
	assert.ErrorContains(t, err, expectedErr.Error())

	mockCbr.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCode(t *testing.T) {
	ctx := context.Background()
	service, _, mockRepo, _, _ := setupTestService()

	charCode := "usd"
	expected := &entity.Currency{CharCode: "USD", Value: 90.5}

	mockRepo.On("GetRateByCharCode", ctx, "USD").Return(expected, nil)

	result, err := service.GetRateByCharCode(ctx, charCode)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCode_NotFound(t *testing.T) {
	ctx := context.Background()
	service, _, mockRepo, _, _ := setupTestService()

	charCode := "USD"

	mockRepo.On("GetRateByCharCode", ctx, charCode).Return((*entity.Currency)(nil), postgres.ErrNotFound)

	_, err := service.GetRateByCharCode(ctx, charCode)
	assert.ErrorContains(t, err, "not found")

	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCodeAndDate_FutureDate(t *testing.T) {
	ctx := context.Background()
	service, _, _, _, _ := setupTestService()

	futureDate := time.Now().Add(24 * time.Hour)
	_, err := service.GetRateByCharCodeAndDate(ctx, "USD", futureDate)
	assert.ErrorContains(t, err, "cannot fetch rates for future dates")
}

func TestGetRateByCharCodeAndDate_Today(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, mockRepo, _, _ := setupTestService()

	charCode := "USD"
	today := time.Now().Truncate(24 * time.Hour)
	cbrDateStr := today.Format("02/01/2006")

	sampleResp := &cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "USD", Name: "US Dollar", Nominal: 1, Value: "90.5", NumCode: "840"},
		},
		Date: today.Format("02.01.2006"),
	}

	mockCbr.On("FetchRates", ctx, cbrDateStr).Return(sampleResp, nil)

	rates, err := convertCBRResponse(*sampleResp)
	require.NoError(t, err)

	mockRepo.On("StoreHistoricalRates", ctx, today, mock.MatchedBy(func(r []entity.Currency) bool {
		return assert.ElementsMatch(t, rates, r)
	})).Return(nil)

	result, err := service.GetRateByCharCodeAndDate(ctx, charCode, today)
	assert.NoError(t, err)
	assert.Equal(t, rates[0].Value, result.Value)

	mockCbr.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCodeAndDate_PastDate_FromDB(t *testing.T) {
	ctx := context.Background()
	service, _, mockRepo, _, _ := setupTestService()

	charCode := "USD"
	pastDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	dateStr := pastDate.Format("2006-01-02")
	expected := &entity.Currency{CharCode: "USD", Value: 90.5, Date: pastDate}

	mockRepo.On("GetRateByCharCodeAndDate", ctx, "USD", dateStr).Return(expected, nil)

	result, err := service.GetRateByCharCodeAndDate(ctx, charCode, pastDate)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCodeAndDate_PastDate_FetchFromCBR(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, mockRepo, _, _ := setupTestService()

	charCode := "USD"
	pastDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	dateStr := pastDate.Format("2006-01-02")
	cbrDateStr := pastDate.Format("02/01/2006")

	mockRepo.On("GetRateByCharCodeAndDate", ctx, "USD", dateStr).Return((*entity.Currency)(nil), postgres.ErrNotFound)

	sampleResp := &cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "USD", Name: "US Dollar", Nominal: 1, Value: "90.5", NumCode: "840"},
		},
		Date: pastDate.Format("02.01.2006"),
	}

	mockCbr.On("FetchRates", ctx, cbrDateStr).Return(sampleResp, nil)

	rates, err := convertCBRResponse(*sampleResp)
	require.NoError(t, err)

	mockRepo.On("StoreHistoricalRates", ctx, pastDate, mock.MatchedBy(func(r []entity.Currency) bool {
		return assert.ElementsMatch(t, rates, r)
	})).Return(nil)

	result, err := service.GetRateByCharCodeAndDate(ctx, charCode, pastDate)
	assert.NoError(t, err)
	assert.Equal(t, rates[0].Value, result.Value)

	mockCbr.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestGetRateByCharCodeAndDate_PastDate_NotFoundInCBR(t *testing.T) {
	ctx := context.Background()
	service, mockCbr, mockRepo, _, _ := setupTestService()

	charCode := "USD"
	pastDate := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	dateStr := pastDate.Format("2006-01-02")
	cbrDateStr := pastDate.Format("02/01/2006")

	mockRepo.On("GetRateByCharCodeAndDate", ctx, "USD", dateStr).Return((*entity.Currency)(nil), postgres.ErrNotFound)

	sampleResp := &cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "EUR", Name: "Euro", Nominal: 1, Value: "100.2", NumCode: "978"},
		},
		Date: pastDate.Format("02.01.2006"),
	}

	mockCbr.On("FetchRates", ctx, cbrDateStr).Return(sampleResp, nil)

	rates, err := convertCBRResponse(*sampleResp)
	require.NoError(t, err)

	mockRepo.On("StoreHistoricalRates", ctx, pastDate, mock.MatchedBy(func(r []entity.Currency) bool {
		return assert.ElementsMatch(t, rates, r)
	})).Return(nil)

	_, err = service.GetRateByCharCodeAndDate(ctx, charCode, pastDate)
	assert.ErrorContains(t, err, "not found")

	mockCbr.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestConvertCBRResponse(t *testing.T) {
	now := time.Now()
	sampleResp := cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "USD", Name: "US Dollar", Nominal: 1, Value: "90,5", NumCode: "840"},
			{CharCode: "EUR", Name: "Euro", Nominal: 1, Value: "100.2", NumCode: "978"},
			{CharCode: "INVALID", Name: "Invalid", Nominal: 1, Value: "abc", NumCode: "000"},
			{CharCode: "ZERO", Name: "Zero", Nominal: 1, Value: "0", NumCode: "000"},
		},
		Date: now.Format("02.01.2006"),
	}

	rates, err := convertCBRResponse(sampleResp)
	assert.NoError(t, err)
	assert.Len(t, rates, 2)

	assert.Equal(t, "USD", rates[0].CharCode)
	assert.Equal(t, 90.5, rates[0].Value)
	assert.Equal(t, "EUR", rates[1].CharCode)
	assert.Equal(t, 100.2, rates[1].Value)
}

func TestConvertCBRResponse_NoValutes(t *testing.T) {
	sampleResp := cbr.ValCurs{Valutes: []cbr.Valute{}}
	rates, err := convertCBRResponse(sampleResp)
	assert.NoError(t, err)
	assert.Empty(t, rates)
}

func TestConvertCBRResponse_AllSkipped(t *testing.T) {
	sampleResp := cbr.ValCurs{
		Valutes: []cbr.Valute{
			{CharCode: "INVALID", Name: "Invalid", Nominal: 1, Value: "abc", NumCode: "000"},
		},
	}

	rates, err := convertCBRResponse(sampleResp)
	assert.Error(t, err)
	assert.Empty(t, rates)
}
