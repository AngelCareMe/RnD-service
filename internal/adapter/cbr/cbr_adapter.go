package cbr

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"

	"github.com/sirupsen/logrus"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *logrus.Logger
}

func NewClient(logger *logrus.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
		baseURL: "https://www.cbr.ru/scripts",
		logger:  logger,
	}
}

func (c *Client) FetchRates(ctx context.Context, date string) (*ValCurs, error) {
	url := fmt.Sprintf("%s/XML_daily.asp?date_req=%s", c.baseURL, date)

	c.logger.Infof("Fetching rates from URL: %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Errorf("Failed to create request: %v", err)
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Заголовки для имитации браузера
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Errorf("Failed to fetch by API: %v", err)
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Infof("Response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error read response body: %w", err)
		}
		c.logger.Debugf("Response body length: %d", len(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Errorf("Failed to read response body: %v", err)
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if len(body) == 0 {
		c.logger.Error("Empty response body from CBR")
		return nil, errors.New("empty response body")
	}

	c.logger.Debugf("Response body length: %d bytes", len(body))
	c.logger.Debugf("First 200 chars: %s", string(body)[:min(200, len(body))])

	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		lower := strings.ToLower(charset)
		if lower == "windows-1251" || lower == "cp1251" {
			c.logger.Debugf("Using charset: %s", charset)
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		}
		c.logger.Errorf("Unsupported charset: %s", charset)
		return nil, fmt.Errorf("unsupported charset: %s", charset)
	}

	var valCurs ValCurs
	if err := decoder.Decode(&valCurs); err != nil {
		c.logger.Errorf("Failed to parse XML CBR: %v", err)
		c.logger.Debugf("First 500 chars: %s", string(body)[:min(500, len(body))])
		return nil, fmt.Errorf("parse XML: %w", err)
	}

	c.logger.Infof("Successfully parsed %d currencies", len(valCurs.Valutes))

	c.logger.Infof("Successfully parsed %d currencies", len(valCurs.Valutes))
	if len(valCurs.Valutes) > 0 {
		c.logger.Debugf("First valute: CharCode=%s, Value=%s", valCurs.Valutes[0].CharCode, valCurs.Valutes[0].Value)
	} else {
		c.logger.Warn("No valutes found in parsed response")
	}

	return &valCurs, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
