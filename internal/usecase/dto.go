package usecase

type CurrencyResponse struct {
	CharName string  `json:"char_name" binding:"required,alpha,len=3,uppercase"`
	ValueRUB float64 `json:"value_rub" binding:"required,min=1"`
}
