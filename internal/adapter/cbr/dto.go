package cbr

import "encoding/xml"

type ValCurs struct {
	XMLName xml.Name
	Date    string
	Name    string
	Valutes []Valute
}

type Valute struct {
	ID       string  `xml:"ID,attr"`
	NumCode  int     `xml:"NumCode"`
	CharCode string  `xml:"CharCode"`
	Nominal  int     `xml:"Nominal"`
	Name     string  `xml:"Name"`
	Value    float64 `xml:"Value"`
}
