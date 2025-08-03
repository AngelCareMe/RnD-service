package service

import (
	"RnD-service/internal/entity"
	"context"
	"time"
)

type CurrencyService interface {
	StoreRatesFromCbr(ctx context.Context) error
	GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error)
	GetRateByCharCodeAndDate(ctx context.Context, charCode string, date time.Time) (*entity.Currency, error)
}
