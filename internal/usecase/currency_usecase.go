package usecase

import (
	"RnD-service/internal/service"
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type CurrencyUsecase struct {
	service service.CurrencyService
	logger  *logrus.Logger
}

func NewCurrencyUsecase(service service.CurrencyService, logger *logrus.Logger) *CurrencyUsecase {
	return &CurrencyUsecase{
		service: service,
		logger:  logger,
	}
}

var charCodeRegexp = regexp.MustCompile(`^[A-Z]{3}$`)

func (uc *CurrencyUsecase) FetchAndStoreRatesFromCBR(ctx context.Context) error {
	uc.logger.Info("Fetching rates from API...")
	return uc.service.StoreRatesFromCbr(ctx)
}

func (uc *CurrencyUsecase) GetRateByCharCode(ctx context.Context, charCode string, amount float64) (*CurrencyResponse, error) {
	code := strings.ToUpper(charCode)

	if !charCodeRegexp.MatchString(code) {
		uc.logger.Errorf("Bad Valute format %s", code)
		return nil, errors.New("invalid char code format")
	}

	currency, err := uc.service.GetRateByCharCode(ctx, code)
	if err != nil {
		uc.logger.Errorf("Failed to get rate by char code")
		return nil, err
	}

	convertedValue := (currency.Value / float64(currency.Nominal)) * amount

	result := &CurrencyResponse{
		CharCode: currency.CharCode,
		ValueRUB: convertedValue,
	}

	uc.logger.Infof("Successfuly fetched rate by char code!")

	return result, nil
}

func (uc *CurrencyUsecase) GetHistoricalRateByCharCode(ctx context.Context, charCode string, date time.Time, amount float64) (*CurrencyResponse, error) {
	code := strings.ToUpper(charCode)
	if !charCodeRegexp.MatchString(code) {
		uc.logger.Errorf("Invalid currency code format: %s", code)
		return nil, errors.New("invalid char code format, expected 3 uppercase letters")
	}

	if date.IsZero() {
		date = time.Now().Truncate(24 * time.Hour)
		uc.logger.Debugf("No date provided, using today: %s", date.Format("2006-01-02"))
	}

	today := time.Now().Truncate(24 * time.Hour)
	if date.After(today) {
		uc.logger.Warnf("Requested future date: %s", date.Format("2006-01-02"))
		return nil, errors.New("cannot fetch rates for future dates")
	}

	currency, err := uc.service.GetRateByCharCodeAndDate(ctx, code, date)
	if err != nil {
		uc.logger.WithError(err).Errorf("Failed to get historical rate by char code %s for date %s", code, date.Format("2006-01-02"))
		return nil, err
	}

	convertedValue := (currency.Value / float64(currency.Nominal)) * amount
	result := &CurrencyResponse{
		CharCode: currency.CharCode,
		ValueRUB: convertedValue,
	}
	uc.logger.Infof("Successfully fetched historical rate for %s on %s: %.4f RUB for %.2f unit(s)", currency.CharCode, currency.Date.Format("2006-01-02"), convertedValue, amount)
	return result, nil
}
