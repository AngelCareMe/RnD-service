package service

import (
	"RnD-service/internal/entity"
	"context"
)

type CurrencyService interface {
	StoreRatesFromCbr(ctx context.Context) error
	GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error)
}
