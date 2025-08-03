package entity

import "time"

type Currency struct {
	ID        string    `db:"id" json:"id,omitempty"`
	CharCode  string    `db:"char_code" json:"char_code"`
	Name      string    `db:"name" json:"name,omitempty"`
	Nominal   int       `db:"nominal" json:"nominal,omitempty"`
	Value     float64   `db:"value" json:"value"`
	NumCode   string    `db:"num_code" json:"num_code,omitempty"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at,omitempty"`
	Date      time.Time `db:"date" json:"date,omitempty"`
}
