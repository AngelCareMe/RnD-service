// internal/service/rate_service.go
// Updated to merge variable declaration

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
	"go.uber.org/multierr"
)

type RateService struct {
	cbr    cbr.CbrClient
	dbRepo postgres.PostgresRepository
	logger *logrus.Logger
}

func NewRateService(cbr cbr.CbrClient, dbRepo postgres.PostgresRepository, logger *logrus.Logger) *RateService {
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

func (r *RateService) GetRateByCharCodeAndDate(ctx context.Context, charCode string, date time.Time) (*entity.Currency, error) {
	charCode = strings.ToUpper(charCode)

	requestedDate := date.Truncate(24 * time.Hour)

	today := time.Now().Truncate(24 * time.Hour)
	if requestedDate.After(today) {
		r.logger.Warnf("Requested future date: %s", requestedDate.Format("2006-01-02"))
		return nil, fmt.Errorf("cannot fetch rates for future dates")
	}

	dateStr := requestedDate.Format("2006-01-02")

	if requestedDate.Equal(today) {
		r.logger.Info("Requested rate for today, fetching fresh data from CBR")
		cbrDateStr := requestedDate.Format("02/01/2006")
		r.logger.Infof("Fetching today's currency rates from CBR for date: %s", cbrDateStr)
		resp, err := r.cbr.FetchRates(ctx, cbrDateStr)
		if err != nil {
			r.logger.Errorf("Failed to fetch today's rates from CBR: %v", err)
			return nil, fmt.Errorf("fetch today's rates: %w", err)
		}
		rates, err := convertCBRResponse(*resp)
		if err != nil {
			r.logger.Errorf("Failed to convert historical response for date %s: %v", cbrDateStr, err)
			return nil, fmt.Errorf("convert historical response: %w", err)
		}
		if len(rates) == 0 {
			r.logger.Warnf("No rates found in historical response for date %s", cbrDateStr)
			return nil, fmt.Errorf("no rates available from CBR for date %s", cbrDateStr)
		}
		if err := r.dbRepo.StoreHistoricalRates(ctx, requestedDate, rates); err != nil {
			r.logger.Errorf("Failed to store today's rates in historical DB: %v", err)
		}

		for _, rate := range rates {
			if rate.CharCode == charCode {
				r.logger.Infof("Found today's rate for %s: %.4f", rate.CharCode, rate.Value)
				return &rate, nil
			}
		}
		r.logger.Warnf("Currency code %s not found in today's rates", charCode)
		return nil, fmt.Errorf("currency code %s not found for today", charCode)

	} else if requestedDate.Before(today) {
		r.logger.Infof("Requested rate for past date: %s", dateStr)

		rate, err := r.dbRepo.GetRateByCharCodeAndDate(ctx, charCode, dateStr)
		if err != nil {
			if errors.Is(err, postgres.ErrNotFound) {
				r.logger.Debugf("Historical rate for %s on %s not found in DB, fetching from CBR", charCode, dateStr)
			} else {
				r.logger.WithError(err).Warn("DB error querying historical rate, cannot proceed")
				return nil, err
			}
		} else {
			r.logger.Infof("Found historical rate for %s on %s: %.4f", rate.CharCode, rate.Date, rate.Value)
			return rate, nil
		}

		cbrDateStr := requestedDate.Format("02/01/2006")
		r.logger.Infof("Fetching historical currency rates from CBR for date: %s", cbrDateStr)
		resp, err := r.cbr.FetchRates(ctx, cbrDateStr)
		if err != nil {
			r.logger.Errorf("Failed to fetch historical rates from CBR for date %s: %v", cbrDateStr, err)
			return nil, fmt.Errorf("fetch historical rates from CBR: %w", err)
		}
		rates, err := convertCBRResponse(*resp)
		if err != nil {
			r.logger.Errorf("Failed to convert historical response for date %s: %v", cbrDateStr, err)
			return nil, fmt.Errorf("convert historical response: %w", err)
		}
		if len(rates) == 0 {
			r.logger.Warnf("No rates found in historical response for date %s", cbrDateStr)
			return nil, fmt.Errorf("no rates available from CBR for date %s", cbrDateStr)
		}

		respDate := rates[0].Date
		if !respDate.Equal(requestedDate) {
			r.logger.Warnf("CBR вернул курсы за %s вместо запрошенной %s (возможно, не торговый день)", respDate.Format("2006-01-02"), requestedDate.Format("2006-01-02"))
		}

		if err := r.dbRepo.StoreHistoricalRates(ctx, requestedDate, rates); err != nil {
			r.logger.Errorf("Failed to store historical rates in DB for date %s: %v", dateStr, err)
		}

		for _, rate := range rates {
			if rate.CharCode == charCode {
				r.logger.Infof("Found historical rate for %s on %s: %.4f", rate.CharCode, rate.Date, rate.Value)
				return &rate, nil
			}
		}
		r.logger.Warnf("Currency code %s not found in historical rates for date %s", charCode, dateStr)
		return nil, fmt.Errorf("currency code %s not found for date %s", charCode, dateStr)

	} else {
		r.logger.Warnf("Requested rate for future date: %s", dateStr)
		return nil, fmt.Errorf("cannot fetch rates for future dates")
	}
}

func convertCBRResponse(resp cbr.ValCurs) ([]entity.Currency, error) {
	var result []entity.Currency
	var errs []error

	if len(resp.Valutes) == 0 {
		logrus.Warn("No valutes found in response")
		return result, nil
	}

	var respDate time.Time
	if resp.Date != "" {
		var err error
		respDate, err = time.Parse("02.01.2006", resp.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CBR response date '%s': %w", resp.Date, err)
		}
	} else {
		respDate = time.Now().Truncate(24 * time.Hour)
		logrus.Warn("No date in CBR response, using current date")
	}

	skipped := 0
	for _, valute := range resp.Valutes {
		value, err := valute.GetValue()
		if err != nil {
			logrus.Debugf("Skipped %s due to parse error: %v", valute.CharCode, err)
			skipped++
			continue
		}
		if value == 0 {
			logrus.Debugf("Skipped %s due to zero value", valute.CharCode)
			skipped++
			continue
		}

		rate := entity.Currency{
			CharCode:  valute.CharCode,
			Name:      valute.Name,
			Nominal:   valute.Nominal,
			Value:     value,
			NumCode:   valute.NumCode,
			UpdatedAt: time.Now(),
			Date:      respDate,
		}
		result = append(result, rate)
	}

	logrus.Infof("Converted %d valid rates out of %d (skipped %d)", len(result), len(resp.Valutes), skipped)

	if len(result) == 0 && len(resp.Valutes) > 0 {
		errs = append(errs, fmt.Errorf("all %d valutes were skipped", len(resp.Valutes)))
	}

	if len(errs) > 0 {
		return result, multierr.Combine(errs...)
	}
	return result, nil
}
