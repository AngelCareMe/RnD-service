package postgres

import (
	"RnD-service/internal/entity"
	"context"
)

type PostgresRepository interface {
	StoreRates(ctx context.Context, rates []entity.Currency) error
	GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error)
}
