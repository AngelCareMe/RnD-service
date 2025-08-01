package usecase

import (
	"RnD-service/internal/service"
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

type CurrencyUsecase struct {
	service service.RateService
	logger  *logrus.Logger
}

func NewCurrencyUsecase(service service.RateService, logger *logrus.Logger) *CurrencyUsecase {
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
		CharName: currency.CharCode,
		ValueRUB: convertedValue,
	}

	uc.logger.Infof("Successfuly fetched rate by char code!")

	return result, nil
}
