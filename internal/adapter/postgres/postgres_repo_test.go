package postgres

import (
	"context"
	"errors"
	"io"
	"regexp"
	"testing"
	"time"

	"RnD-service/internal/entity"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepo(t *testing.T) (*PostgresRepo, pgxmock.PgxPoolIface) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	repo := NewPostgresRepo(mock, logger)
	return repo, mock
}

func TestGetRateByCharCode(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "USD"
	expected := &entity.Currency{
		CharCode: charCode,
		Name:     "US Dollar",
		Nominal:  1,
		Value:    90.5,
	}

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value").
		From("currency_rates").
		Where(squirrel.Eq{"char_code": charCode}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnRows(pgxmock.NewRows([]string{"char_code", "name", "nominal", "value"}).
			AddRow(expected.CharCode, expected.Name, expected.Nominal, expected.Value))

	result, err := repo.GetRateByCharCode(ctx, charCode)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRateByCharCode_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "USD"

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value").
		From("currency_rates").
		Where(squirrel.Eq{"char_code": charCode}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetRateByCharCode(ctx, charCode)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRateByCharCode_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "USD"

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value").
		From("currency_rates").
		Where(squirrel.Eq{"char_code": charCode}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	expectedErr := errors.New("database error")
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnError(expectedErr)

	result, err := repo.GetRateByCharCode(ctx, charCode)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, expectedErr.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreRates(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	now := time.Now().UTC()
	rates := []entity.Currency{
		{
			CharCode:  "USD",
			Name:      "US Dollar",
			Nominal:   1,
			Value:     90.5,
			NumCode:   "840",
			UpdatedAt: now,
		},
		{
			CharCode:  "EUR",
			Name:      "Euro",
			Nominal:   1,
			Value:     100.2,
			NumCode:   "978",
			UpdatedAt: now,
		},
	}

	mock.ExpectBegin()

	eb := mock.ExpectBatch()

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
		require.NoError(t, err)

		eb.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(args...).
			WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
	}

	mock.ExpectCommit()

	err := repo.StoreRates(ctx, rates)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreRates_ErrorInBatch(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	now := time.Now().UTC()
	rates := []entity.Currency{
		{
			CharCode:  "USD",
			Name:      "US Dollar",
			Nominal:   1,
			Value:     90.5,
			NumCode:   "840",
			UpdatedAt: now,
		},
		{
			CharCode:  "EUR",
			Name:      "Euro",
			Nominal:   1,
			Value:     100.2,
			NumCode:   "978",
			UpdatedAt: now,
		},
	}

	mock.ExpectBegin()

	eb := mock.ExpectBatch()

	// First insert succeeds
	query1, args1, err := psql.Insert("currency_rates").
		Columns("char_code", "name", "nominal", "value", "num_code", "updated_at").
		Values(rates[0].CharCode, rates[0].Name, rates[0].Nominal, rates[0].Value, rates[0].NumCode, rates[0].UpdatedAt).
		Suffix(`
                ON CONFLICT (char_code) DO UPDATE SET
                    name = EXCLUDED.name,
                    nominal = EXCLUDED.nominal,
                    value = EXCLUDED.value,
                    num_code = EXCLUDED.num_code,
                    updated_at = EXCLUDED.updated_at
            `).
		ToSql()
	require.NoError(t, err)

	eb.ExpectExec(regexp.QuoteMeta(query1)).
		WithArgs(args1...).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	// Second insert fails
	query2, args2, err := psql.Insert("currency_rates").
		Columns("char_code", "name", "nominal", "value", "num_code", "updated_at").
		Values(rates[1].CharCode, rates[1].Name, rates[1].Nominal, rates[1].Value, rates[1].NumCode, rates[1].UpdatedAt).
		Suffix(`
                ON CONFLICT (char_code) DO UPDATE SET
                    name = EXCLUDED.name,
                    nominal = EXCLUDED.nominal,
                    value = EXCLUDED.value,
                    num_code = EXCLUDED.num_code,
                    updated_at = EXCLUDED.updated_at
            `).
		ToSql()
	require.NoError(t, err)

	expectedErr := errors.New("insert error")
	eb.ExpectExec(regexp.QuoteMeta(query2)).
		WithArgs(args2...).
		WillReturnError(expectedErr)

	mock.ExpectRollback()

	err = repo.StoreRates(ctx, rates)
	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedErr.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRateByCharCodeAndDate(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "usd" // will be uppercased
	dateStr := "2025-08-02"
	date, err := time.Parse("2006-01-02", dateStr)
	require.NoError(t, err)

	expected := &entity.Currency{
		CharCode: "USD",
		Name:     "US Dollar",
		Nominal:  1,
		Value:    90.5,
		Date:     date,
	}

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value", "date").
		From("historical_currency_rates").
		Where(squirrel.Eq{"char_code": "USD", "date": dateStr}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnRows(pgxmock.NewRows([]string{"char_code", "name", "nominal", "value", "date"}).
			AddRow(expected.CharCode, expected.Name, expected.Nominal, expected.Value, expected.Date))

	result, err := repo.GetRateByCharCodeAndDate(ctx, charCode, dateStr)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRateByCharCodeAndDate_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "USD"
	dateStr := "2025-08-02"

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value", "date").
		From("historical_currency_rates").
		Where(squirrel.Eq{"char_code": "USD", "date": dateStr}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnError(pgx.ErrNoRows)

	result, err := repo.GetRateByCharCodeAndDate(ctx, charCode, dateStr)
	assert.Nil(t, result)
	assert.Equal(t, ErrNotFound, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRateByCharCodeAndDate_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	charCode := "USD"
	dateStr := "2025-08-02"

	query, args, err := psql.
		Select("char_code", "name", "nominal", "value", "date").
		From("historical_currency_rates").
		Where(squirrel.Eq{"char_code": "USD", "date": dateStr}).
		Limit(1).
		ToSql()
	require.NoError(t, err)

	expectedErr := errors.New("database error")
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(args...).
		WillReturnError(expectedErr)

	result, err := repo.GetRateByCharCodeAndDate(ctx, charCode, dateStr)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, expectedErr.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreHistoricalRates(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	date := time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)
	rates := []entity.Currency{
		{
			CharCode: "USD",
			Name:     "US Dollar",
			Nominal:  1,
			Value:    90.5,
			NumCode:  "840",
		},
		{
			CharCode: "EUR",
			Name:     "Euro",
			Nominal:  1,
			Value:    100.2,
			NumCode:  "978",
		},
	}

	mock.ExpectBegin()

	eb := mock.ExpectBatch()

	for _, rate := range rates {
		query, args, err := psql.Insert("historical_currency_rates").
			Columns("char_code", "date", "name", "nominal", "value", "num_code").
			Values(rate.CharCode, date, rate.Name, rate.Nominal, rate.Value, rate.NumCode).
			Suffix("ON CONFLICT (char_code, date) DO NOTHING").
			ToSql()
		require.NoError(t, err)

		eb.ExpectExec(regexp.QuoteMeta(query)).
			WithArgs(args...).
			WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))
	}

	mock.ExpectCommit()

	err := repo.StoreHistoricalRates(ctx, date, rates)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreHistoricalRates_ErrorInBatch(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	date := time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)
	rates := []entity.Currency{
		{
			CharCode: "USD",
			Name:     "US Dollar",
			Nominal:  1,
			Value:    90.5,
			NumCode:  "840",
		},
		{
			CharCode: "EUR",
			Name:     "Euro",
			Nominal:  1,
			Value:    100.2,
			NumCode:  "978",
		},
	}

	mock.ExpectBegin()

	eb := mock.ExpectBatch()

	// First insert succeeds
	query1, args1, err := psql.Insert("historical_currency_rates").
		Columns("char_code", "date", "name", "nominal", "value", "num_code").
		Values(rates[0].CharCode, date, rates[0].Name, rates[0].Nominal, rates[0].Value, rates[0].NumCode).
		Suffix("ON CONFLICT (char_code, date) DO NOTHING").
		ToSql()
	require.NoError(t, err)

	eb.ExpectExec(regexp.QuoteMeta(query1)).
		WithArgs(args1...).
		WillReturnResult(pgconn.NewCommandTag("INSERT 0 1"))

	// Second insert fails
	query2, args2, err := psql.Insert("historical_currency_rates").
		Columns("char_code", "date", "name", "nominal", "value", "num_code").
		Values(rates[1].CharCode, date, rates[1].Name, rates[1].Nominal, rates[1].Value, rates[1].NumCode).
		Suffix("ON CONFLICT (char_code, date) DO NOTHING").
		ToSql()
	require.NoError(t, err)

	expectedErr := errors.New("insert error")
	eb.ExpectExec(regexp.QuoteMeta(query2)).
		WithArgs(args2...).
		WillReturnError(expectedErr)

	mock.ExpectRollback()

	err = repo.StoreHistoricalRates(ctx, date, rates)
	assert.Error(t, err)
	assert.ErrorContains(t, err, expectedErr.Error())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreHistoricalRates_EmptyRates(t *testing.T) {
	ctx := context.Background()
	repo, mock := setupTestRepo(t)
	defer mock.Close()

	date := time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)
	rates := []entity.Currency{}

	// No expectations since early return
	err := repo.StoreHistoricalRates(ctx, date, rates)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
