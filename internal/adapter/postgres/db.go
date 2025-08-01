package postgres

import (
	"RnD-service/pkg/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func InitDBPool(cfg config.Config, logger *logrus.Logger) (*pgxpool.Pool, error) {
	dsn := BuildDSN(cfg)

	const maxRetries = 5
	var pool *pgxpool.Pool
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < maxRetries; i++ {
		logger.Infof("DB connection attempt #%d", i+1)

		pool, err = pgxpool.New(ctx, dsn)
		if err != nil {
			logger.Warnf("failed to create DB pool on attempt #%d: %v", i+1, err)
		} else {
			err = pool.Ping(ctx)
			if err == nil {
				logger.Infof("successfully connected to DB on attempt #%d", i+1)
				return pool, nil
			}
			logger.Warnf("failed to ping DB on attempt #%d: %v", i+1, err)
			pool.Close()
		}

		if i < maxRetries-1 {
			sleepDuration := time.Second * time.Duration(i+1)
			logger.Infof("waiting %s before next attempt", sleepDuration)
			time.Sleep(sleepDuration)
		}
	}

	logger.Errorf("Failed to create and ping DB pool after %d attempts: %v", maxRetries, err)

	return nil, fmt.Errorf("failed to create and ping DB pool after %d retries: %w", maxRetries, err)
}
