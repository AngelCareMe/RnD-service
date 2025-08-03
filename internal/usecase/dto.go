package usecase

type CurrencyResponse struct {
	CharCode string  `json:"char_name"`
	ValueRUB float64 `json:"value_rub"`
}
