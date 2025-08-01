package handler

type CurrencyRequest struct {
	CharName string  `json:"char_name" binding:"required,alpha,len=3,uppercase"`
	Value    float64 `json:"value" binding:"required,min=1"`
}
