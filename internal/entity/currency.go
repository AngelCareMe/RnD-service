package entity

type Currency struct {
	ID        string  `db:"id" json:"id,omitempty"`
	CharCode  string  `db:"char_code" json:"char_code"`
	Name      string  `db:"name" json:"name"`
	Nominal   int     `db:"nominal" json:"nominal"`
	Value     float64 `db:"value" json:"value"`
	NumCode   int     `db:"num_code" json:"num_code"`
	UpdatedAt string  `db:"updated_at"  json:"updated_at,omitempty"`
}

type CurrencyRequest struct {
	CharName string  `json:"char_name" binding:"required,alpha,len=3,uppercase"`
	Value    float64 `json:"value" binding:"required,min=1"`
}

type CurrencyResponse struct {
	CharName string  `json:"char_name" binding:"required,alpha,len=3,uppercase"`
	ValueRUB float64 `json:"value_rub" binding:"required,min=1"`
}
