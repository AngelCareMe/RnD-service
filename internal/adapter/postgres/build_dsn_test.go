// adapter/postgres/build_dsn_test.go
// New file for testing BuildDSN in postgres package

package postgres

import (
	"RnD-service/pkg/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDSN(t *testing.T) {
	cfg := config.Config{
		Postgres: struct {
			Host     string `mapstructure:"host"`
			Port     string `mapstructure:"port"`
			DBName   string `mapstructure:"dbname"`
			User     string `mapstructure:"user"`
			Password string `mapstructure:"password"`
			SSLMode  string `mapstructure:"sslmode"`
		}{
			Host:     "localhost",
			Port:     "5432",
			DBName:   "testdb",
			User:     "testuser",
			Password: "testpass",
			SSLMode:  "disable",
		},
	}

	dsn := BuildDSN(cfg)
	assert.Equal(t, "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable", dsn)
}
