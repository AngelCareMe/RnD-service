package postgres

import (
	"RnD-service/internal/entity"
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

var (
	psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
)

type PostgresRepo struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
}

func NewPostgresRepo(pool *pgxpool.Pool, logger *logrus.Logger) *PostgresRepo {
	return &PostgresRepo{
		pool:   pool,
		logger: logger,
	}
}

func (r *PostgresRepo) StoreRates(ctx context.Context, rates []entity.Currency) error {
	r.logger.Info("Start storing currency rates")

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.WithError(err).Error("Failed to begin transaction")
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, rate := range rates {
		query, args, err := psql.Insert("currency_rates").
			Columns("id", "char_code", "name", "nominal", "value", "NumCode", "updated_at").
			Values(rate.ID, rate.CharCode, rate.Name, rate.Nominal, rate.Value, rate.NumCode, rate.UpdatedAt).
			Suffix(`
				ON CONFLICT (char_code) DO UPDATE SET
					name = EXCLUDED.name,
					nominal = EXCLUDED.nominal,
					value = EXCLUDED.value,
					num_code = EXCLUDED.num_code,
					updated_at = EXCLUDED.updated_at
			`).
			ToSql()
		if err != nil {
			r.logger.WithError(err).WithField("rate", rate.CharCode).Error("Failed to build insert query")
			return fmt.Errorf("build insert: %w", err)
		}

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			r.logger.WithError(err).WithField("rate", rate.CharCode).Error("Failed to execute insert query")
			return fmt.Errorf("exec insert: %w", err)
		}

		r.logger.WithFields(logrus.Fields{
			"char_code": rate.CharCode,
			"name":      rate.Name,
			"value":     rate.Value,
			"nominal":   rate.Nominal,
		}).Info("Stored currency rate")
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.WithError(err).Error("Failed to commit transaction")
		return err
	}

	r.logger.Info("Successfully stored all currency rates")
	return nil
}

func (r *PostgresRepo) GetRateByCharCode(ctx context.Context, charCode string) (*entity.Currency, error) {
	r.logger.WithField("char_code", charCode).Info("Getting currency rate by char code")

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value").
		From("currency_rates").
		Where(sq.Eq{"char_code": charCode}).
		Limit(1).
		ToSql()

	if err != nil {
		r.logger.WithError(err).Error("Failed to build select query")
		return nil, fmt.Errorf("build select: %w", err)
	}

	var rate entity.Currency
	err = r.pool.QueryRow(ctx, query, args...).
		Scan(
			&rate.CharCode,
			&rate.Name,
			&rate.Nominal,
			&rate.Value,
		)
	if err != nil {
		r.logger.WithError(err).WithField("char_code", charCode).Error("Failed to query currency rate")
		return nil, fmt.Errorf("query scan: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"char_code": rate.CharCode,
		"name":      rate.Name,
		"value":     rate.Value,
		"nominal":   rate.Nominal,
	}).Info("Successfully retrieved currency rate")

	return &rate, nil
}
