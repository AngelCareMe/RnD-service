package usecase

import (
	"RnD-service/internal/entity"
	"context"
)

type RateUsecase interface {
	FetchAndStoreRates(ctx context.Context) error
	GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error)
}
