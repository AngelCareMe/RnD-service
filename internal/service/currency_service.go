package service

import (
	"RnD-service/internal/adapter/cbr"
	"RnD-service/internal/adapter/postgres"
	"RnD-service/internal/entity"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type RateService struct {
	cbr    cbr.CbrClient
	dbRepo postgres.PostgresRepo
	logger *logrus.Logger
}

func NewRateService(cbr cbr.CbrClient, dbRepo postgres.PostgresRepo, logger *logrus.Logger) *RateService {
	return &RateService{
		cbr:    cbr,
		dbRepo: dbRepo,
		logger: logger,
	}
}

func (r *RateService) StoreRatesFromCbr(ctx context.Context) error {
	date := time.Now().Format("02/01/2006")
	r.logger.Info("Fetching currency rates from CBR...")

	resp, err := r.cbr.FetchRates(ctx, date)
	if err != nil {
		r.logger.Errorf("Failed to fetch rates from CBR: %v", err)
		return fmt.Errorf("fetch rates: %w", err)
	}

	rates, err := convertCBRResponse(*resp)
	if err != nil {
		r.logger.Errorf("Failed to convert response: %v", err)
		return fmt.Errorf("convert response: %w", err)
	}

	if len(rates) == 0 {
		r.logger.Warn("No rates found in response")
		return errors.New("no rates to store")
	}

	r.logger.Infof("Storing %d rates for date %s", len(rates), date)

	if err := r.dbRepo.StoreRates(ctx, rates); err != nil {
		r.logger.Errorf("Failed to store rates in DB: %v", err)
		return fmt.Errorf("store rates in DB: %w", err)
	}

	r.logger.Info("Currency rates successfully stored.")
	return nil
}

func (r *RateService) GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error) {
	r.logger.Infof("Fetching currency by CharCode: %s", charCode)

	charCode = strings.ToUpper(charCode)

	rate, err := r.dbRepo.GetRateByCharCode(ctx, charCode)
	if err != nil {
		r.logger.Errorf("Failed to get currency rate for %s: %v", charCode, err)
		return nil, fmt.Errorf("get rate by char code: %w", err)
	}

	if rate == nil {
		r.logger.Warnf("No currency found for CharCode: %s", charCode)
		return nil, fmt.Errorf("valute code %s not found", charCode)
	}

	r.logger.Infof("Found rate for %s: %.4f", rate.CharCode, rate.Value)
	return rate, nil
}

func convertCBRResponse(resp cbr.ValCurs) ([]entity.Currency, error) {
	var result []entity.Currency

	logrus.Infof("Converting %d valutes", len(resp.Valutes))

	for i, valute := range resp.Valutes {

		if i == 0 {
			logrus.Debugf("First valute debug: %+v", valute)
		}

		logrus.Debugf("Processing valute: ID=%s, CharCode=%s, Value=%s",
			valute.ID, valute.CharCode, valute.Value)

		if len(resp.Valutes) == 0 {
			logrus.Warn("No valutes found in response")
			return []entity.Currency{}, nil
		}

		value, err := valute.GetValue()
		if err != nil {
			logrus.Errorf("Failed to parse value for %s: %s, error: %v",
				valute.CharCode, valute.Value, err)
			continue
		}

		if value == 0 {
			continue
		}

		rate := entity.Currency{
			ID:        valute.ID,
			CharCode:  valute.CharCode,
			Name:      valute.Name,
			Nominal:   valute.Nominal,
			Value:     value,
			NumCode:   valute.NumCode,
			UpdatedAt: resp.Date,
		}

		result = append(result, rate)
	}

	logrus.Infof("Converted %d valid rates", len(result))
	return result, nil
}
