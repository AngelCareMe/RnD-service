package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"RnD-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRateUsecase struct {
	mock.Mock
}

func (m *mockRateUsecase) FetchAndStoreRatesFromCBR(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRateUsecase) GetHistoricalRateByCharCode(ctx context.Context, charCode string, date time.Time, amount float64) (*usecase.CurrencyResponse, error) {
	args := m.Called(ctx, charCode, date, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CurrencyResponse), args.Error(1)
}

func (m *mockRateUsecase) GetRateByCharCode(ctx context.Context, charCode string, amount float64) (*usecase.CurrencyResponse, error) {
	args := m.Called(ctx, charCode, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CurrencyResponse), args.Error(1)
}

func setupTestHandler() (*CurrencyHandler, *mockRateUsecase, *logrus.Logger, *test.Hook) {
	mockUsecase := new(mockRateUsecase)
	logger, hook := test.NewNullLogger()
	handler := NewRateHandler(mockUsecase, logger)
	return handler, mockUsecase, logger, hook
}

func TestStoreRatesFromCBR_Success(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	mockUsecase.On("FetchAndStoreRatesFromCBR", mock.Anything).Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Initialize the Request field to avoid nil pointer dereference
	c.Request, _ = http.NewRequest("GET", "/", nil)

	handler.StoreRatesFromCBR(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Rates successfully updated", response["message"])

	mockUsecase.AssertExpectations(t)
}

func TestStoreRatesFromCBR_Error(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	expectedErr := errors.New("usecase error")
	mockUsecase.On("FetchAndStoreRatesFromCBR", mock.Anything).Return(expectedErr)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Initialize the Request field
	c.Request, _ = http.NewRequest("GET", "/", nil)

	handler.StoreRatesFromCBR(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Failed to fetch rates", response["error"])

	mockUsecase.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_MissingVal(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Initialize the Request field
	c.Request, _ = http.NewRequest("GET", "/", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "missing required query parameter 'val'")
}

func TestGetHistoricalRateByCharCode_InvalidDate(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD&date=invalid", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "invalid date format")
}

func TestGetHistoricalRateByCharCode_InvalidAmount(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD&amount=invalid", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "invalid 'amount' parameter")
}

func TestGetHistoricalRateByCharCode_FutureDate(t *testing.T) {
	handler, _, _, _ := setupTestHandler()

	futureDate := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD&date="+futureDate, nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "cannot fetch rates for future dates")
}

func TestGetHistoricalRateByCharCode_UsecaseError(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	expectedErr := errors.New("usecase error")
	mockUsecase.On("GetHistoricalRateByCharCode", mock.Anything, "USD", mock.AnythingOfType("time.Time"), 1.0).Return((*usecase.CurrencyResponse)(nil), expectedErr)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, expectedErr.Error(), response["error"])

	mockUsecase.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_NotFoundError(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	expectedErr := errors.New("not found")
	mockUsecase.On("GetHistoricalRateByCharCode", mock.Anything, "USD", mock.AnythingOfType("time.Time"), 1.0).Return((*usecase.CurrencyResponse)(nil), expectedErr)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockUsecase.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_Success(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	expectedResponse := &usecase.CurrencyResponse{
		CharCode: "USD",
		ValueRUB: 90.5,
	}
	mockUsecase.On("GetHistoricalRateByCharCode", mock.Anything, "USD", mock.AnythingOfType("time.Time"), 1.0).Return(expectedResponse, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response usecase.CurrencyResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, expectedResponse, &response)

	mockUsecase.AssertExpectations(t)
}

func TestGetHistoricalRateByCharCode_WithAmountAndDate(t *testing.T) {
	handler, mockUsecase, _, _ := setupTestHandler()

	date := time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC)
	amount := 2.0
	expectedResponse := &usecase.CurrencyResponse{
		CharCode: "USD",
		ValueRUB: 181.0,
	}
	mockUsecase.On("GetHistoricalRateByCharCode", mock.Anything, "USD", date, amount).Return(expectedResponse, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?val=USD&amount=2&date=2025-08-01", nil)

	handler.GetHistoricalRateByCharCode(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response usecase.CurrencyResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, expectedResponse, &response)

	mockUsecase.AssertExpectations(t)
}
