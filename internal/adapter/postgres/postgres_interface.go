package postgres

import (
	"RnD-service/internal/entity"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository interface {
	StoreRates(ctx context.Context, rates []entity.Currency) error
	GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error)

	StoreHistoricalRates(ctx context.Context, date time.Time, rates []entity.Currency) error
	GetRateByCharCodeAndDate(ctx context.Context, charCode, date string) (*entity.Currency, error)
}

type Pool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}
