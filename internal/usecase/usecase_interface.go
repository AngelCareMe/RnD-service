package usecase

import (
	"context"
	"time"
)

type RateUsecase interface {
	FetchAndStoreRatesFromCBR(ctx context.Context) error
	GetRateByCharCode(ctx context.Context, charCode string, amount float64) (*CurrencyResponse, error)
	GetHistoricalRateByCharCode(ctx context.Context, charCode string, date time.Time, amount float64) (*CurrencyResponse, error)
}
