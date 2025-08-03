// internal/adapter/cbr/valute_test.go
// Update tests for GetValue and XML

package cbr

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValute_GetValue_Success(t *testing.T) {
	v := Valute{Value: "90,1234"}
	value, err := v.GetValue()
	require.NoError(t, err)
	assert.Equal(t, 90.1234, value)
}

func TestValute_GetValue_CommaReplacement(t *testing.T) {
	v := Valute{Value: "1234,56"}
	value, err := v.GetValue()
	require.NoError(t, err)
	assert.Equal(t, 1234.56, value)
}

func TestValute_GetValue_Invalid(t *testing.T) {
	v := Valute{Value: "invalid"}
	_, err := v.GetValue()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "strconv.ParseFloat")
}

func TestValCurs_XMLUnmarshal(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="windows-1251"?>
	<ValCurs Date="02.01.2006" name="Foreign Currency Market">
		<Valute ID="R01235">
			<NumCode>840</NumCode>
			<CharCode>USD</CharCode>
			<Nominal>1</Nominal>
			<Name>US Dollar</Name>
			<Value>90,1234</Value>
			<VunitRate>90,1234</VunitRate>
		</Valute>
	</ValCurs>`

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlData)))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if charset == "windows-1251" {
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		}
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}

	var vc ValCurs
	err := decoder.Decode(&vc)
	require.NoError(t, err)
	assert.Equal(t, "02.01.2006", vc.Date)
	assert.Equal(t, "Foreign Currency Market", vc.Name)
	assert.Len(t, vc.Valutes, 1)
	assert.Equal(t, "R01235", vc.Valutes[0].ID)
	assert.Equal(t, "840", vc.Valutes[0].NumCode)
	assert.Equal(t, "USD", vc.Valutes[0].CharCode)
	assert.Equal(t, 1, vc.Valutes[0].Nominal)
	assert.Equal(t, "US Dollar", vc.Valutes[0].Name)
	assert.Equal(t, "90,1234", vc.Valutes[0].Value)
	assert.Equal(t, "90,1234", vc.Valutes[0].VunitRate)
}

func TestValCurs_XMLMarshal(t *testing.T) {
	vc := ValCurs{
		Date: "02.01.2006",
		Name: "Foreign Currency Market",
		Valutes: []Valute{
			{
				ID:        "R01235",
				NumCode:   "840",
				CharCode:  "USD",
				Nominal:   1,
				Name:      "US Dollar",
				Value:     "90,1234",
				VunitRate: "90,1234",
			},
		},
	}

	data, err := xml.Marshal(vc)
	require.NoError(t, err)
	assert.Contains(t, string(data), `<ValCurs Date="02.01.2006" name="Foreign Currency Market">`)
	assert.Contains(t, string(data), `<Valute ID="R01235">`)
	assert.Contains(t, string(data), `<NumCode>840</NumCode>`)
	assert.Contains(t, string(data), `<CharCode>USD</CharCode>`)
	assert.Contains(t, string(data), `<Nominal>1</Nominal>`)
	assert.Contains(t, string(data), `<Name>US Dollar</Name>`)
	assert.Contains(t, string(data), `<Value>90,1234</Value>`)
	assert.Contains(t, string(data), `<VunitRate>90,1234</VunitRate>`)
}
