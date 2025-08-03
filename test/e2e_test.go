package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"RnD-service/internal/adapter/cbr"
	projectpostgres "RnD-service/internal/adapter/postgres"
	"RnD-service/internal/handler"
	"RnD-service/internal/service"
	"RnD-service/internal/usecase"
	"RnD-service/pkg/config"
	"RnD-service/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type mockCbrClient struct {
	logger *logrus.Logger
}

func (m *mockCbrClient) FetchRates(ctx context.Context, date string) (*cbr.ValCurs, error) {
	m.logger.Infof("Mock FetchRates called for date: %s", date)
	if date == "03/08/2025" || date == "12/01/2023" {
		return &cbr.ValCurs{
			Date: "12.01.2023",
			Name: "Foreign Currency Market",
			Valutes: []cbr.Valute{
				{
					ID:        "R01235",
					NumCode:   "840",
					CharCode:  "USD",
					Nominal:   1,
					Name:      "US Dollar",
					Value:     "69,0202",
					VunitRate: "69,0202",
				},
			},
		}, nil
	}
	return nil, fmt.Errorf("no data for date %s", date)
}

func TestE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start Postgres container
	pgContainer, err := testpostgres.Run(
		ctx,
		"postgres:15-alpine",
		testpostgres.WithDatabase("currency"),
		testpostgres.WithUsername("postgres"),
		testpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		pgContainer.Terminate(context.Background())
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create config (mocked)
	cfg := &config.Config{}
	cfg.Log.Level = "debug"
	cfg.Postgres.SSLMode = "disable" // not used since we use dsn directly

	log := logger.Init(cfg.Log.Level)

	// Init DB pool with test dsn
	poolConfig, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)
	t.Cleanup(func() {
		dbPool.Close()
	})

	// Run migrations (execute CREATE TABLE statements)
	conn, err := dbPool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS currency_rates (
		    char_code   VARCHAR(3) PRIMARY KEY,
		    name        TEXT        NOT NULL,
		    nominal     INTEGER     NOT NULL CHECK (nominal > 0),
		    value       NUMERIC(20, 4) NOT NULL CHECK (value >= 0),
		    num_code    VARCHAR(3),
		    updated_at  TIMESTAMP   NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS uniq_currency_char_code ON currency_rates(char_code);
	`)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS historical_currency_rates (
		    char_code   VARCHAR(3) NOT NULL,
		    date        DATE NOT NULL,
		    name        TEXT        NOT NULL,
		    nominal     INTEGER     NOT NULL CHECK (nominal > 0),
		    value       NUMERIC(20, 4) NOT NULL CHECK (value >= 0),
		    num_code    VARCHAR(3),
		    PRIMARY KEY (char_code, date)
		);
		CREATE INDEX IF NOT EXISTS idx_historical_currency_date ON historical_currency_rates(date);
		CREATE INDEX IF NOT EXISTS idx_historical_currency_char_code ON historical_currency_rates(char_code);
	`)
	require.NoError(t, err)

	// Init adapters
	cbrClient := &mockCbrClient{logger: log}
	dbRepo := projectpostgres.NewPostgresRepo(dbPool, log)

	// Init service
	currencyService := service.NewRateService(cbrClient, dbRepo, log)

	// Init usecase
	currencyUsecase := usecase.NewCurrencyUsecase(currencyService, log)

	// Init handler
	currencyHandler := handler.NewRateHandler(currencyUsecase, log)

	// Setup Gin router
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "http://127.0.0.1:8081"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	r.GET("/currency/rates", currencyHandler.StoreRatesFromCBR)
	r.GET("/currency/rate", currencyHandler.GetHistoricalRateByCharCode)

	// Start server in goroutine
	srv := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()
	t.Cleanup(func() {
		srv.Shutdown(context.Background())
	})

	// Wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:8081/nonexistent")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusNotFound
	}, 5*time.Second, 100*time.Millisecond)

	t.Run("StoreRatesFromCBR", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/currency/rates")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]string
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "Rates successfully updated", result["message"])
	})

	t.Run("GetHistoricalRateByCharCode", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/currency/rate?val=USD&date=2023-01-12")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			CharName string  `json:"char_name"`
			ValueRUB float64 `json:"value_rub"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "USD", result.CharName)
		assert.InDelta(t, 69.0202, result.ValueRUB, 0.0001)
	})

	t.Run("GetHistoricalRateByCharCode_InvalidDate", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/currency/rate?val=USD&date=invalid")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
