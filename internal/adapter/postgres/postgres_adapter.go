package postgres

import (
	"RnD-service/internal/entity"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

var (
	psql        = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ErrNotFound = errors.New("not found")
)

type PostgresRepo struct {
	pool   Pool
	logger *logrus.Logger
}

func NewPostgresRepo(pool Pool, logger *logrus.Logger) *PostgresRepo {
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

	batch := &pgx.Batch{}
	for _, rate := range rates {
		query, args, err := psql.Insert("currency_rates").
			Columns("char_code", "name", "nominal", "value", "num_code", "updated_at").
			Values(rate.CharCode, rate.Name, rate.Nominal, rate.Value, rate.NumCode, rate.UpdatedAt).
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
			return fmt.Errorf("build insert for %s: %w", rate.CharCode, err)
		}
		batch.Queue(query, args...)
	}

	br := tx.SendBatch(ctx, batch)

	var batchErrs error
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			batchErrs = multierr.Append(batchErrs, err)
			r.logger.WithError(err).Errorf("Failed batch exec for rate %d", i)
		}
	}

	if err := br.Close(); err != nil {
		batchErrs = multierr.Append(batchErrs, err)
		r.logger.WithError(err).Error("Failed to close batch results")
	}

	if batchErrs != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			r.logger.WithError(rbErr).Error("Failed to rollback tx after batch errors")
		}
		return fmt.Errorf("batch exec/close errors: %w", batchErrs)
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.WithError(err).Error("Failed to commit tx")
		return fmt.Errorf("commit tx: %w", err)
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.logger.WithError(err).WithFields(logrus.Fields{"char_code": charCode}).Error("Failed to query historical rate")
		return nil, fmt.Errorf("query historical rate: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"char_code": rate.CharCode,
		"name":      rate.Name,
		"value":     rate.Value,
		"nominal":   rate.Nominal,
	}).Info("Successfully retrieved currency rate")

	return &rate, nil
}

func (r *PostgresRepo) StoreHistoricalRates(ctx context.Context, date time.Time, rates []entity.Currency) error {
	r.logger.WithField("date", date.Format("2006-01-02")).Info("Start storing historical currency rates")

	if len(rates) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.WithError(err).Error("Failed to begin transaction for historical rates")
		return fmt.Errorf("begin tx: %w", err)
	}

	batch := &pgx.Batch{}
	for _, rate := range rates {
		query, args, err := psql.Insert("historical_currency_rates").
			Columns("char_code", "date", "name", "nominal", "value", "num_code").
			Values(rate.CharCode, date, rate.Name, rate.Nominal, rate.Value, rate.NumCode).
			Suffix("ON CONFLICT (char_code, date) DO NOTHING").
			ToSql()
		if err != nil {
			return fmt.Errorf("build insert for %s on %s: %w", rate.CharCode, date, err)
		}
		batch.Queue(query, args...)
	}

	br := tx.SendBatch(ctx, batch)

	var batchErrs error
	var inserted int64
	for i := 0; i < batch.Len(); i++ {
		ct, err := br.Exec()
		if err != nil {
			batchErrs = multierr.Append(batchErrs, err)
			r.logger.WithError(err).Errorf("Failed batch exec for historical rate %d", i)
		} else {
			inserted += ct.RowsAffected()
		}
	}

	if err := br.Close(); err != nil {
		batchErrs = multierr.Append(batchErrs, err)
		r.logger.WithError(err).Error("Failed to close batch results for historical")
	}

	if batchErrs != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			r.logger.WithError(rbErr).Error("Failed to rollback historical tx")
		}
		return fmt.Errorf("batch exec/close errors for historical: %w", batchErrs)
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.WithError(err).Error("Failed to commit historical tx")
		return fmt.Errorf("commit tx: %w", err)
	}

	r.logger.WithField("date", date.Format("2006-01-02")).Infof("Successfully stored %d historical rates", inserted)
	return nil
}

func (r *PostgresRepo) GetRateByCharCodeAndDate(ctx context.Context, charCode, date string) (*entity.Currency, error) {
	r.logger.WithFields(logrus.Fields{"char_code": charCode, "date": date}).Info("Getting historical currency rate by char code and date")
	query, args, err := psql.
		Select("char_code", "name", "nominal", "value", "date").
		From("historical_currency_rates").
		Where(sq.Eq{"char_code": strings.ToUpper(charCode), "date": date}).
		Limit(1).
		ToSql()
	if err != nil {
		r.logger.WithError(err).Error("Failed to build select query for historical rate")
		return nil, fmt.Errorf("build select: %w", err)
	}
	var rate entity.Currency
	err = r.pool.QueryRow(ctx, query, args...).
		Scan(
			&rate.CharCode,
			&rate.Name,
			&rate.Nominal,
			&rate.Value,
			&rate.Date,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.WithFields(logrus.Fields{"char_code": charCode, "date": date}).Debug("Historical rate not found in DB")
			return nil, ErrNotFound
		}
		r.logger.WithError(err).WithFields(logrus.Fields{"char_code": charCode, "date": date}).Error("Failed to query historical rate")
		return nil, fmt.Errorf("query scan: %w", err)
	}
	r.logger.WithFields(logrus.Fields{
		"char_code": rate.CharCode,
		"name":      rate.Name,
		"value":     rate.Value,
		"nominal":   rate.Nominal,
		"date":      rate.Date,
	}).Info("Successfully retrieved historical currency rate")
	return &rate, nil
}
