package handler

type GetRateRequest struct {
	CharCode string  `json:"char_code" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
}
