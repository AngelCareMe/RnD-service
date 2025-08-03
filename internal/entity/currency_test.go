package entity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrency_MarshalJSON(t *testing.T) {
	now := time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)
	currency := Currency{
		ID:        "123",
		CharCode:  "USD",
		Name:      "US Dollar",
		Nominal:   1,
		Value:     90.5,
		NumCode:   "840",
		UpdatedAt: now,
		Date:      now,
	}

	data, err := json.Marshal(currency)
	require.NoError(t, err)

	expected := `{"id":"123","char_code":"USD","name":"US Dollar","nominal":1,"value":90.5,"num_code":"840","updated_at":"2025-08-02T00:00:00Z","date":"2025-08-02T00:00:00Z"}`
	assert.JSONEq(t, expected, string(data))
}

func TestCurrency_MarshalJSON_OmitEmpty(t *testing.T) {
	currency := Currency{
		CharCode: "USD",
		Value:    90.5,
	}

	data, err := json.Marshal(currency)
	require.NoError(t, err)

	expected := `{"char_code":"USD","value":90.5,"updated_at":"0001-01-01T00:00:00Z","date":"0001-01-01T00:00:00Z"}`
	assert.JSONEq(t, expected, string(data))
}

func TestCurrency_UnmarshalJSON(t *testing.T) {
	jsonData := `{"id":"123","char_code":"USD","name":"US Dollar","nominal":1,"value":90.5,"num_code":"840","updated_at":"2025-08-02T00:00:00Z","date":"2025-08-02T00:00:00Z"}`

	var currency Currency
	err := json.Unmarshal([]byte(jsonData), &currency)
	require.NoError(t, err)

	now := time.Date(2025, 8, 2, 0, 0, 0, 0, time.UTC)
	expected := Currency{
		ID:        "123",
		CharCode:  "USD",
		Name:      "US Dollar",
		Nominal:   1,
		Value:     90.5,
		NumCode:   "840",
		UpdatedAt: now,
		Date:      now,
	}
	assert.Equal(t, expected, currency)
}

func TestCurrency_UnmarshalJSON_Partial(t *testing.T) {
	jsonData := `{"char_code":"USD","value":90.5}`

	var currency Currency
	err := json.Unmarshal([]byte(jsonData), &currency)
	require.NoError(t, err)

	expected := Currency{
		CharCode: "USD",
		Value:    90.5,
	}
	assert.Equal(t, expected, currency)
}

func TestCurrency_MarshalJSON_ZeroValues(t *testing.T) {
	currency := Currency{}

	data, err := json.Marshal(currency)
	require.NoError(t, err)

	expected := `{"char_code":"","value":0,"updated_at":"0001-01-01T00:00:00Z","date":"0001-01-01T00:00:00Z"}`
	assert.JSONEq(t, expected, string(data))
}
